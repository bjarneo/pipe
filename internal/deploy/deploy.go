package deploy

import (
	"context"
	"fmt"
	"strings"

	"github.com/bjarneo/pipe/internal/config"
	"github.com/bjarneo/pipe/internal/docker"
	"github.com/bjarneo/pipe/internal/logger"
	"github.com/bjarneo/pipe/internal/ssh"
	"github.com/bjarneo/pipe/internal/stats"
)

func Deploy(cfg *config.Config, log *logger.Logger) error {
	return DeployWithContext(context.Background(), cfg, log)
}

func DeployWithContext(ctx context.Context, cfg *config.Config, log *logger.Logger) error {
	if cfg.JSONOutput {
		log.SetQuiet(true)
	}

	st := stats.New()
	st.SetImageInfo(cfg.Image, cfg.Tag)
	st.SetContainerInfo(cfg.ContainerName, cfg.Host)

	if err := cfg.Validate(); err != nil {
		return err
	}

	if cfg.DryRun {
		log.Step("DRY RUN - no changes will be made")
		return printDryRunSummary(cfg, log)
	}

	if err := docker.Check(ctx, cfg, log); err != nil {
		return err
	}
	if err := ssh.CheckContext(ctx, cfg, log); err != nil {
		return err
	}

	if err := docker.Build(ctx, cfg, log); err != nil {
		return err
	}
	log.StepProgress(1, 4, "Building image", "done")

	transferResult, err := docker.Transfer(ctx, cfg, log, st)
	if err != nil {
		return err
	}
	log.StepProgress(2, 4, "Analyzing layers", fmt.Sprintf("%d changed, %d cached", transferResult.NewLayers, transferResult.CachedLayers))
	log.StepProgress(3, 4, "Transferring delta", fmt.Sprintf("%s (saved %s)", stats.FormatBytes(transferResult.TransferredBytes), stats.FormatBytes(transferResult.ImageSize-transferResult.TransferredBytes)))

	if cfg.EnvFile != "" {
		if err := copyEnvFile(ctx, cfg, log); err != nil {
			return err
		}
	}

	if err := docker.ReplaceContainer(ctx, cfg, log); err != nil {
		return err
	}
	log.StepProgress(4, 4, "Starting container", "running")

	if len(cfg.RemoteCommands) > 0 {
		if err := executeRemoteCommands(ctx, cfg, log); err != nil {
			return err
		}
	}

	if cfg.JSONOutput {
		jsonOutput, err := st.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to generate JSON output: %w", err)
		}
		fmt.Println(jsonOutput)
	}

	return nil
}

func printDryRunSummary(cfg *config.Config, log *logger.Logger) error {
	fmt.Println("=== DRY RUN SUMMARY ===")
	fmt.Printf("Target: %s@%s (port %s)\n", cfg.User, cfg.Host, cfg.SSHPort)
	fmt.Printf("Image: %s:%s\n", cfg.Image, cfg.Tag)
	fmt.Printf("Container: %s\n", cfg.ContainerName)
	fmt.Printf("Ports: %s -> %s\n", cfg.HostPort, cfg.ContainerPort)
	fmt.Printf("Platform: %s\n", cfg.Platform)
	fmt.Printf("Restart Policy: %s\n", cfg.RestartPolicy)

	if cfg.Network != "" {
		fmt.Printf("Network: %s\n", cfg.Network)
	}
	if cfg.CPUs != "" {
		fmt.Printf("CPUs: %s\n", cfg.CPUs)
	}
	if cfg.Memory != "" {
		fmt.Printf("Memory: %s\n", cfg.Memory)
	}
	if cfg.EnvFile != "" {
		fmt.Printf("Env File: %s\n", cfg.EnvFile)
	}
	if len(cfg.Env) > 0 {
		fmt.Printf("Environment Variables: %d\n", len(cfg.Env))
	}
	if len(cfg.Volumes) > 0 {
		fmt.Printf("Volumes: %v\n", cfg.Volumes)
	}
	if len(cfg.Labels) > 0 {
		fmt.Printf("Labels: %d\n", len(cfg.Labels))
	}
	if cfg.HealthCmd != "" {
		fmt.Printf("Health Check: %s\n", cfg.HealthCmd)
	}
	if len(cfg.BuildArgs) > 0 {
		fmt.Printf("Build Args: %d\n", len(cfg.BuildArgs))
	}
	if len(cfg.RemoteCommands) > 0 {
		fmt.Printf("Remote Commands: %d\n", len(cfg.RemoteCommands))
		for i, cmd := range cfg.RemoteCommands {
			fmt.Printf("  [%d] %s\n", i+1, cmd)
		}
	}
	if cfg.Privileged {
		fmt.Println("Privileged: true")
	}
	if cfg.ReadOnly {
		fmt.Println("Read-Only: true")
	}
	if cfg.Init {
		fmt.Println("Init: true")
	}

	fmt.Println("=== END DRY RUN ===")
	return nil
}

func getCurrentContainerImage(ctx context.Context, cfg *config.Config, log *logger.Logger) (string, error) {
	cmd := fmt.Sprintf("%s \"docker inspect --format='{{.Config.Image}}' %s\"",
		ssh.BuildSSHCommand(cfg), cfg.ContainerName)
	result, err := ssh.ExecuteCommandContext(ctx, log, cmd, "Getting current container information")
	if err != nil {
		return "", fmt.Errorf("failed to get current container information: %w", err)
	}
	return strings.TrimSpace(result.Stdout), nil
}

func getImageHistory(ctx context.Context, cfg *config.Config, log *logger.Logger) ([]string, error) {
	cmd := fmt.Sprintf("%s \"docker images %s --format '{{.Repository}}:{{.Tag}}___{{.CreatedAt}}' | sort -k2 -r\"",
		ssh.BuildSSHCommand(cfg), cfg.Image)
	result, err := ssh.ExecuteCommandContext(ctx, log, cmd, "Getting image history")
	if err != nil {
		return nil, fmt.Errorf("failed to get image history: %w", err)
	}
	return strings.Split(strings.TrimSpace(result.Stdout), "\n"), nil
}

func findPreviousImage(images []string, currentImage string) (string, error) {
	if len(images) < 2 {
		return "", fmt.Errorf("no previous version found to rollback to")
	}

	for i, img := range images {
		parts := strings.Split(img, "___")
		if len(parts) == 0 {
			continue
		}
		imageName := parts[0]
		if imageName == currentImage && i+1 < len(images) {
			nextParts := strings.Split(images[i+1], "___")
			if len(nextParts) > 0 {
				return nextParts[0], nil
			}
		}
	}
	return "", fmt.Errorf("could not find previous version to rollback to")
}

func verifyContainerRunning(ctx context.Context, cfg *config.Config, log *logger.Logger) (bool, error) {
	cmd := fmt.Sprintf("%s \"docker ps --filter name=%s --format '{{.Status}}'\"",
		ssh.BuildSSHCommand(cfg), cfg.ContainerName)
	result, err := ssh.ExecuteCommandContext(ctx, log, cmd, "Verifying container status")
	if err != nil {
		return false, err
	}
	return strings.Contains(result.Stdout, "Up"), nil
}

func cleanupBackupContainer(ctx context.Context, cfg *config.Config, log *logger.Logger) {
	cmd := fmt.Sprintf("%s \"docker rm %s_backup\"", ssh.BuildSSHCommand(cfg), cfg.ContainerName)
	_, _ = ssh.ExecuteCommandContext(ctx, log, cmd, "Cleaning up backup container")
}

func Rollback(cfg *config.Config, log *logger.Logger) error {
	return RollbackWithContext(context.Background(), cfg, log)
}

func RollbackWithContext(ctx context.Context, cfg *config.Config, log *logger.Logger) error {
	log.Step("Starting rollback")

	if err := cfg.Validate(); err != nil {
		return err
	}

	log.Step("Checking SSH connectivity")
	if err := ssh.CheckContext(ctx, cfg, log); err != nil {
		return err
	}

	log.Step("Finding previous version")
	currentImage, err := getCurrentContainerImage(ctx, cfg, log)
	if err != nil {
		return err
	}

	images, err := getImageHistory(ctx, cfg, log)
	if err != nil {
		return err
	}

	previousImage, err := findPreviousImage(images, currentImage)
	if err != nil {
		return err
	}

	log.Step(fmt.Sprintf("Rolling back to %s", previousImage))

	if err := performRollback(ctx, cfg, log, previousImage); err != nil {
		return err
	}

	cleanupBackupContainer(ctx, cfg, log)

	fmt.Println("Rollback completed successfully!")
	return nil
}

func copyEnvFile(ctx context.Context, cfg *config.Config, log *logger.Logger) error {
	copyEnvCmd := fmt.Sprintf("%s %s %s@%s:~/%s",
		ssh.BuildSCPCommand(cfg), cfg.EnvFile, cfg.User, cfg.Host, cfg.EnvFile)
	_, err := ssh.ExecuteCommandContext(ctx, log, copyEnvCmd, "Copying environment file to server")
	return err
}

func executeRemoteCommands(ctx context.Context, cfg *config.Config, log *logger.Logger) error {
	for i, cmd := range cfg.RemoteCommands {
		if cmd == "" {
			continue
		}
		remoteCmd := fmt.Sprintf("%s \"%s\"", ssh.BuildSSHCommand(cfg), cmd)
		description := fmt.Sprintf("Executing remote command [%d/%d]: %s", i+1, len(cfg.RemoteCommands), cmd)
		if _, err := ssh.ExecuteCommandContext(ctx, log, remoteCmd, description); err != nil {
			return fmt.Errorf("remote command failed: %w", err)
		}
	}
	return nil
}

func buildRollbackCommands(cfg *config.Config, previousImage string) string {
	containerArgs := docker.NewRunBuilder(cfg).BuildWithImage(previousImage)

	commands := []string{
		fmt.Sprintf("docker stop %s", cfg.ContainerName),
		fmt.Sprintf("docker rename %s %s_backup", cfg.ContainerName, cfg.ContainerName),
		fmt.Sprintf("docker run %s", strings.Join(containerArgs, " ")),
	}
	return strings.Join(commands, " && ")
}

func performRollback(ctx context.Context, cfg *config.Config, log *logger.Logger, previousImage string) error {
	rollbackCommands := buildRollbackCommands(cfg, previousImage)

	rollbackCmd := fmt.Sprintf("%s \"%s\"", ssh.BuildSSHCommand(cfg), rollbackCommands)
	if _, err := ssh.ExecuteCommandContext(ctx, log, rollbackCmd, "Rolling back to previous version"); err != nil {
		if restoreErr := restoreBackup(ctx, cfg, log); restoreErr != nil {
			return fmt.Errorf("rollback failed and restore failed: %w (original error: %v)", restoreErr, err)
		}
		return fmt.Errorf("rollback failed, restored previous version: %w", err)
	}

	running, err := verifyContainerRunning(ctx, cfg, log)
	if err != nil {
		return err
	}

	if !running {
		if restoreErr := restoreBackup(ctx, cfg, log); restoreErr != nil {
			return fmt.Errorf("rollback verification failed and restore failed: %w", restoreErr)
		}
		return fmt.Errorf("rollback verification failed, restored previous version")
	}

	return nil
}

func restoreBackup(ctx context.Context, cfg *config.Config, log *logger.Logger) error {
	restoreCmd := fmt.Sprintf("%s \"docker stop %s || true && docker rm %s || true && docker rename %s_backup %s && docker start %s\"",
		ssh.BuildSSHCommand(cfg), cfg.ContainerName, cfg.ContainerName,
		cfg.ContainerName, cfg.ContainerName, cfg.ContainerName)
	_, err := ssh.ExecuteCommandContext(ctx, log, restoreCmd, "Restoring previous version after failed rollback")
	return err
}
