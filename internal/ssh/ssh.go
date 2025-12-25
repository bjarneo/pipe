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

// =============================================================================
// Types
// =============================================================================

// CommandResult contains the output of a command
type CommandResult struct {
	Stdout string
	Stderr string
}

// Executor defines the interface for command execution
type Executor interface {
	Execute(command, description string) (*CommandResult, error)
	ExecuteContext(ctx context.Context, command, description string) (*CommandResult, error)
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
func (e *DefaultExecutor) Execute(command, description string) (*CommandResult, error) {
	return ExecuteCommand(e.Log, command, description)
}

// ExecuteContext runs a command with context support
func (e *DefaultExecutor) ExecuteContext(ctx context.Context, command, description string) (*CommandResult, error) {
	return ExecuteCommandContext(ctx, e.Log, command, description)
}

// =============================================================================
// SSH Command Building
// =============================================================================

// GetCommand returns the full SSH command string
func GetCommand(cfg *config.Config) string {
	var parts []string
	parts = append(parts, "ssh")

	if flag := keyFlag(cfg.SSHKey); flag != "" {
		parts = append(parts, flag)
	}
	if flag := portFlag(cfg.SSHPort, "-p"); flag != "" {
		parts = append(parts, flag)
	}

	parts = append(parts, fmt.Sprintf("%s@%s", cfg.User, cfg.Host))
	return strings.Join(parts, " ")
}

// GetSCPCommand returns the SCP command prefix with key and port flags
func GetSCPCommand(cfg *config.Config) string {
	var parts []string
	parts = append(parts, "scp")

	if flag := keyFlag(cfg.SSHKey); flag != "" {
		parts = append(parts, flag)
	}
	if flag := portFlag(cfg.SSHPort, "-P"); flag != "" {
		parts = append(parts, flag)
	}

	return strings.Join(parts, " ")
}

// keyFlag returns the SSH key flag if key is set
func keyFlag(key string) string {
	if key != "" {
		return fmt.Sprintf("-i %s", key)
	}
	return ""
}

// portFlag returns the port flag if non-default
func portFlag(port, flag string) string {
	if port != "" && port != "22" {
		return fmt.Sprintf("%s %s", flag, port)
	}
	return ""
}

// GetKeyFlag returns the SSH key flag if SSHKey is set (exported for compatibility)
func GetKeyFlag(cfg *config.Config) string {
	return keyFlag(cfg.SSHKey)
}

// GetPortFlag returns the SSH port flag if non-default (exported for compatibility)
func GetPortFlag(cfg *config.Config) string {
	return portFlag(cfg.SSHPort, "-p")
}

// =============================================================================
// Connection Check
// =============================================================================

// Check checks SSH connection to the remote host
func Check(cfg *config.Config, log *logger.Logger) error {
	return CheckContext(context.Background(), cfg, log)
}

// CheckContext checks SSH connection with context support
func CheckContext(ctx context.Context, cfg *config.Config, log *logger.Logger) error {
	command := fmt.Sprintf("%s echo \"SSH connection successful\"", GetCommand(cfg))
	_, err := ExecuteCommandContext(ctx, log, command, "Checking SSH connection")
	return err
}

// =============================================================================
// Command Execution
// =============================================================================

// ExecuteCommand executes a shell command and streams the output
func ExecuteCommand(log *logger.Logger, command, description string) (*CommandResult, error) {
	return ExecuteCommandContext(context.Background(), log, command, description)
}

// ExecuteCommandContext executes a shell command with context support
func ExecuteCommandContext(ctx context.Context, log *logger.Logger, command, description string) (*CommandResult, error) {
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

	// Read output streams concurrently
	result, scanErr := readOutputStreams(stdout, stderr, log.IsVerbose())
	if scanErr != nil {
		return nil, scanErr
	}

	if err := cmd.Wait(); err != nil {
		return nil, handleCommandError(ctx, err)
	}

	return result, nil
}

// readOutputStreams reads stdout and stderr concurrently
func readOutputStreams(stdout, stderr interface{ Read([]byte) (int, error) }, verbose bool) (*CommandResult, error) {
	var stdoutBuilder, stderrBuilder strings.Builder
	var mu sync.Mutex
	var wg sync.WaitGroup
	var scanErr error

	wg.Add(2)

	// Read stdout
	go func() {
		defer wg.Done()
		if err := readStream(stdout, &stdoutBuilder, &mu, verbose, false); err != nil {
			mu.Lock()
			if scanErr == nil {
				scanErr = fmt.Errorf("stdout scanner error: %w", err)
			}
			mu.Unlock()
		}
	}()

	// Read stderr
	go func() {
		defer wg.Done()
		if err := readStream(stderr, &stderrBuilder, &mu, verbose, true); err != nil {
			mu.Lock()
			if scanErr == nil {
				scanErr = fmt.Errorf("stderr scanner error: %w", err)
			}
			mu.Unlock()
		}
	}()

	wg.Wait()

	if scanErr != nil {
		return nil, scanErr
	}

	return &CommandResult{
		Stdout: stdoutBuilder.String(),
		Stderr: stderrBuilder.String(),
	}, nil
}

// readStream reads from a stream and appends to the builder
func readStream(r interface{ Read([]byte) (int, error) }, builder *strings.Builder, mu *sync.Mutex, verbose, isStderr bool) error {
	scanner := bufio.NewScanner(r)

	// Use larger buffer for long lines (1MB max)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if verbose {
			if isStderr {
				fmt.Println("STDERR:", line)
			} else {
				fmt.Println(line)
			}
		}

		mu.Lock()
		builder.WriteString(line + "\n")
		mu.Unlock()
	}

	return scanner.Err()
}

// handleCommandError converts command errors to descriptive messages
func handleCommandError(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return fmt.Errorf("command cancelled: %w", ctx.Err())
	}
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() != 0 {
		return fmt.Errorf("command failed with exit code %d: %v", exitErr.ExitCode(), err)
	}
	return fmt.Errorf("command failed: %v", err)
}
