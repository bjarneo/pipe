package deploy

import (
	"fmt"
	"strings"

	"github.com/bjarneo/pipe/internal/config"
	"github.com/bjarneo/pipe/internal/docker"
	"github.com/bjarneo/pipe/internal/logger"
	"github.com/bjarneo/pipe/internal/ssh"
	"github.com/bjarneo/pipe/internal/stats"
)

// Deploy performs the main deployment process
func Deploy(cfg *config.Config, log *logger.Logger) error {
	// Initialize stats tracking
	st := stats.New()
	st.SetImageInfo(cfg.Image, cfg.Tag)
	st.SetContainerInfo(cfg.ContainerName, cfg.Host)

	// Log start of deployment
	if cfg.DryRun {
		log.Step("DRY RUN - no changes will be made")
	} else {
		log.Step(fmt.Sprintf("Deploying %s:%s to %s", cfg.Image, cfg.Tag, cfg.Host))
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return err
	}

	// Dry run: print configuration summary and exit
	if cfg.DryRun {
		return printDryRunSummary(cfg, log)
	}

	// Preliminary checks
	log.Step("Checking Docker and SSH connectivity")
	if err := docker.Check(cfg, log); err != nil {
		return err
	}

	if err := ssh.Check(cfg, log); err != nil {
		return err
	}

	// Build Docker image
	log.Step("Building Docker image")
	if err := docker.Build(cfg, log); err != nil {
		return err
	}

	// Transfer Docker image
	log.Step("Transferring image to remote host")
	if err := docker.Transfer(cfg, log, st); err != nil {
		return err
	}

	// Copy environment file if it exists
	if cfg.EnvFile != "" {
		log.Step("Copying environment file")
		if err := copyEnvFile(cfg, log); err != nil {
			return err
		}
	}

	// Deploy container
	log.Step("Starting container")
	if err := docker.Deploy(cfg, log); err != nil {
		return err
	}

	// Execute remote commands if specified
	if len(cfg.RemoteCommands) > 0 {
		log.Step("Running post-deployment commands")
		if err := executeRemoteCommands(cfg, log); err != nil {
			return err
		}
	}

	// Print deployment summary
	st.PrintSummary()

	fmt.Println("Deployment completed successfully!")
	return nil
}

// printDryRunSummary prints what would happen during deployment
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

// getCurrentContainerImage retrieves the image used by the current container
func getCurrentContainerImage(cfg *config.Config, log *logger.Logger) (string, error) {
	cmd := fmt.Sprintf("%s \"docker inspect --format='{{.Config.Image}}' %s\"",
		ssh.GetCommand(cfg), cfg.ContainerName)
	result, err := ssh.ExecuteCommand(log, cmd, "Getting current container information")
	if err != nil {
		return "", fmt.Errorf("failed to get current container information: %w", err)
	}
	return strings.TrimSpace(result.Stdout), nil
}

// getImageHistory retrieves the list of images sorted by creation time
func getImageHistory(cfg *config.Config, log *logger.Logger) ([]string, error) {
	cmd := fmt.Sprintf("%s \"docker images %s --format '{{.Repository}}:{{.Tag}}___{{.CreatedAt}}' | sort -k2 -r\"",
		ssh.GetCommand(cfg), cfg.Image)
	result, err := ssh.ExecuteCommand(log, cmd, "Getting image history")
	if err != nil {
		return nil, fmt.Errorf("failed to get image history: %w", err)
	}
	return strings.Split(strings.TrimSpace(result.Stdout), "\n"), nil
}

// findPreviousImage finds the previous image version from the history
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

// verifyContainerRunning checks if a container is running
func verifyContainerRunning(cfg *config.Config, log *logger.Logger) (bool, error) {
	cmd := fmt.Sprintf("%s \"docker ps --filter name=%s --format '{{.Status}}'\"",
		ssh.GetCommand(cfg), cfg.ContainerName)
	result, err := ssh.ExecuteCommand(log, cmd, "Verifying container status")
	if err != nil {
		return false, err
	}
	return strings.Contains(result.Stdout, "Up"), nil
}

// cleanupBackupContainer removes the backup container
func cleanupBackupContainer(cfg *config.Config, log *logger.Logger) {
	cmd := fmt.Sprintf("%s \"docker rm %s_backup\"", ssh.GetCommand(cfg), cfg.ContainerName)
	_, _ = ssh.ExecuteCommand(log, cmd, "Cleaning up backup container")
}

// Rollback performs a rollback to the previous version
func Rollback(cfg *config.Config, log *logger.Logger) error {
	log.Step("Starting rollback")

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return err
	}

	// Check SSH connection
	log.Step("Checking SSH connectivity")
	if err := ssh.Check(cfg, log); err != nil {
		return err
	}

	// Get current container image
	log.Step("Finding previous version")
	currentImage, err := getCurrentContainerImage(cfg, log)
	if err != nil {
		return err
	}

	// Get image history
	images, err := getImageHistory(cfg, log)
	if err != nil {
		return err
	}

	// Find previous image
	previousImage, err := findPreviousImage(images, currentImage)
	if err != nil {
		return err
	}

	log.Step(fmt.Sprintf("Rolling back to %s", previousImage))

	// Perform the rollback
	if err := performRollback(cfg, log, previousImage); err != nil {
		return err
	}

	// Clean up backup container
	cleanupBackupContainer(cfg, log)

	fmt.Println("Rollback completed successfully!")
	return nil
}

// copyEnvFile copies the environment file to the remote host
func copyEnvFile(cfg *config.Config, log *logger.Logger) error {
	copyEnvCmd := fmt.Sprintf("%s %s %s@%s:~/%s",
		ssh.GetSCPCommand(cfg), cfg.EnvFile, cfg.User, cfg.Host, cfg.EnvFile)
	_, err := ssh.ExecuteCommand(log, copyEnvCmd, "Copying environment file to server")
	return err
}

// executeRemoteCommands runs custom commands on the remote host after deployment
func executeRemoteCommands(cfg *config.Config, log *logger.Logger) error {
	for i, cmd := range cfg.RemoteCommands {
		if cmd == "" {
			continue
		}
		remoteCmd := fmt.Sprintf("%s \"%s\"", ssh.GetCommand(cfg), cmd)
		description := fmt.Sprintf("Executing remote command [%d/%d]: %s", i+1, len(cfg.RemoteCommands), cmd)
		if _, err := ssh.ExecuteCommand(log, remoteCmd, description); err != nil {
			return fmt.Errorf("remote command failed: %w", err)
		}
	}
	return nil
}

// buildRollbackCommands creates the command string for rollback operation
func buildRollbackCommands(cfg *config.Config, previousImage string) string {
	envFileFlag := ""
	if cfg.EnvFile != "" {
		envFileFlag = fmt.Sprintf("--env-file ~/%s", cfg.EnvFile)
	}

	commands := []string{
		// Stop and rename current container (for backup)
		fmt.Sprintf("docker stop %s", cfg.ContainerName),
		fmt.Sprintf("docker rename %s %s_backup", cfg.ContainerName, cfg.ContainerName),
		// Start container with previous version
		fmt.Sprintf("docker run -d --name %s --restart unless-stopped -p %s:%s %s %s",
			cfg.ContainerName, cfg.HostPort, cfg.ContainerPort,
			envFileFlag, previousImage),
	}
	return strings.Join(commands, " && ")
}

// performRollback executes the rollback operation
func performRollback(cfg *config.Config, log *logger.Logger, previousImage string) error {
	rollbackCommands := buildRollbackCommands(cfg, previousImage)

	// Execute rollback
	rollbackCmd := fmt.Sprintf("%s \"%s\"", ssh.GetCommand(cfg), rollbackCommands)
	if _, err := ssh.ExecuteCommand(log, rollbackCmd, "Rolling back to previous version"); err != nil {
		// If rollback fails, attempt to restore the backup
		if restoreErr := restoreBackup(cfg, log); restoreErr != nil {
			return fmt.Errorf("rollback failed and restore failed: %w (original error: %v)", restoreErr, err)
		}
		return fmt.Errorf("rollback failed, restored previous version: %w", err)
	}

	// Verify new container is running
	running, err := verifyContainerRunning(cfg, log)
	if err != nil {
		return err
	}

	if !running {
		// If verification fails, attempt to restore the backup
		if restoreErr := restoreBackup(cfg, log); restoreErr != nil {
			return fmt.Errorf("rollback verification failed and restore failed: %w", restoreErr)
		}
		return fmt.Errorf("rollback verification failed, restored previous version")
	}

	return nil
}

// restoreBackup attempts to restore the backup container
func restoreBackup(cfg *config.Config, log *logger.Logger) error {
	restoreCmd := fmt.Sprintf("%s \"docker stop %s || true && docker rm %s || true && docker rename %s_backup %s && docker start %s\"",
		ssh.GetCommand(cfg), cfg.ContainerName, cfg.ContainerName,
		cfg.ContainerName, cfg.ContainerName, cfg.ContainerName)
	_, err := ssh.ExecuteCommand(log, restoreCmd, "Restoring previous version after failed rollback")
	return err
} 