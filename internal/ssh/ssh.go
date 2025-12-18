package ssh

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/bjarneo/pipe/internal/config"
	"github.com/bjarneo/pipe/internal/logger"
)

// CommandResult contains the output of a command
type CommandResult struct {
	Stdout string
	Stderr string
}

// Executor defines the interface for command execution
// This allows for mocking in tests and alternative implementations
type Executor interface {
	// Execute runs a command and returns the result
	Execute(command string, description string) (*CommandResult, error)
	// ExecuteContext runs a command with context support
	ExecuteContext(ctx context.Context, command string, description string) (*CommandResult, error)
}

// DefaultExecutor implements Executor using the real shell
type DefaultExecutor struct {
	Log *logger.Logger
}

// NewExecutor creates a new DefaultExecutor
func NewExecutor(log *logger.Logger) *DefaultExecutor {
	return &DefaultExecutor{Log: log}
}

// Execute runs a command using the default executor
func (e *DefaultExecutor) Execute(command string, description string) (*CommandResult, error) {
	return ExecuteCommand(e.Log, command, description)
}

// ExecuteContext runs a command with context support using the default executor
func (e *DefaultExecutor) ExecuteContext(ctx context.Context, command string, description string) (*CommandResult, error) {
	return ExecuteCommandContext(ctx, e.Log, command, description)
}

// GetKeyFlag returns the SSH key flag if SSHKey is set
func GetKeyFlag(cfg *config.Config) string {
	if cfg.SSHKey != "" {
		return fmt.Sprintf("-i %s", cfg.SSHKey)
	}
	return ""
}

// GetPortFlag returns the SSH port flag if non-default
func GetPortFlag(cfg *config.Config) string {
	if cfg.SSHPort != "" && cfg.SSHPort != "22" {
		return fmt.Sprintf("-p %s", cfg.SSHPort)
	}
	return ""
}

// GetCommand returns the full SSH command with or without the key flag
func GetCommand(cfg *config.Config) string {
	var parts []string
	parts = append(parts, "ssh")

	if keyFlag := GetKeyFlag(cfg); keyFlag != "" {
		parts = append(parts, keyFlag)
	}
	if portFlag := GetPortFlag(cfg); portFlag != "" {
		parts = append(parts, portFlag)
	}

	parts = append(parts, fmt.Sprintf("%s@%s", cfg.User, cfg.Host))
	return strings.Join(parts, " ")
}

// GetSCPCommand returns the SCP command prefix with key and port flags
func GetSCPCommand(cfg *config.Config) string {
	var parts []string
	parts = append(parts, "scp")

	if keyFlag := GetKeyFlag(cfg); keyFlag != "" {
		parts = append(parts, keyFlag)
	}
	if cfg.SSHPort != "" && cfg.SSHPort != "22" {
		parts = append(parts, fmt.Sprintf("-P %s", cfg.SSHPort))
	}

	return strings.Join(parts, " ")
}

// Check checks SSH connection to the remote host
func Check(cfg *config.Config, log *logger.Logger) error {
	return CheckContext(context.Background(), cfg, log)
}

// CheckContext checks SSH connection to the remote host with context support
func CheckContext(ctx context.Context, cfg *config.Config, log *logger.Logger) error {
	command := fmt.Sprintf("%s echo \"SSH connection successful\"", GetCommand(cfg))
	_, err := ExecuteCommandContext(ctx, log, command, "Checking SSH connection")
	return err
}

// ExecuteCommand executes a shell command and streams the output
// Deprecated: Use ExecuteCommandContext for better cancellation support
func ExecuteCommand(log *logger.Logger, command string, description string) (*CommandResult, error) {
	return ExecuteCommandContext(context.Background(), log, command, description)
}

// ExecuteCommandContext executes a shell command with context support for cancellation and timeouts
func ExecuteCommandContext(ctx context.Context, log *logger.Logger, command string, description string) (*CommandResult, error) {
	if err := log.Debug(fmt.Sprintf("Executing: %s", command)); err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", command)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %v", err)
	}

	var stdoutBuilder, stderrBuilder strings.Builder
	var mu sync.Mutex
	var wg sync.WaitGroup
	var scanErr error // Capture any scanner errors
	wg.Add(2)

	verbose := log.IsVerbose()

	// Read stdout in real-time
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		// Increase buffer size to handle long lines (1MB max)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if verbose {
				fmt.Println(line)
			}
			mu.Lock()
			stdoutBuilder.WriteString(line + "\n")
			mu.Unlock()
		}
		if err := scanner.Err(); err != nil {
			mu.Lock()
			if scanErr == nil {
				scanErr = fmt.Errorf("stdout scanner error: %w", err)
			}
			mu.Unlock()
		}
	}()

	// Read stderr in real-time
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		// Increase buffer size to handle long lines (1MB max)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "error") || strings.Contains(line, "Error") {
				if verbose {
					fmt.Println("ERROR:", line)
				}
				mu.Lock()
				stderrBuilder.WriteString(line + "\n")
				mu.Unlock()
			} else {
				if verbose {
					fmt.Println(line)
				}
				mu.Lock()
				stdoutBuilder.WriteString(line + "\n")
				mu.Unlock()
			}
		}
		if err := scanner.Err(); err != nil {
			mu.Lock()
			if scanErr == nil {
				scanErr = fmt.Errorf("stderr scanner error: %w", err)
			}
			mu.Unlock()
		}
	}()

	// Wait for both goroutines to finish reading before calling cmd.Wait()
	wg.Wait()

	// Check for scanner errors
	if scanErr != nil {
		return nil, scanErr
	}

	if err := cmd.Wait(); err != nil {
		// Check if context was cancelled or timed out
		if ctx.Err() != nil {
			return nil, fmt.Errorf("command cancelled: %w", ctx.Err())
		}
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() != 0 {
			return nil, fmt.Errorf("command failed with exit code %d: %v", exitErr.ExitCode(), err)
		}
		return nil, fmt.Errorf("command failed: %v", err)
	}

	result := &CommandResult{
		Stdout: stdoutBuilder.String(),
		Stderr: stderrBuilder.String(),
	}

	return result, nil
} 