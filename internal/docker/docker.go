package docker

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/bjarneo/pipe/internal/config"
	"github.com/bjarneo/pipe/internal/logger"
	"github.com/bjarneo/pipe/internal/ssh"
	"github.com/bjarneo/pipe/internal/stats"
)

// Check checks if Docker is installed and running locally and remotely
func Check(cfg *config.Config, log *logger.Logger) error {
	// Check local Docker
	if _, err := ssh.ExecuteCommand(log, "docker info", "Checking local Docker installation"); err != nil {
		return fmt.Errorf("local Docker check failed: %v", err)
	}

	// Check remote Docker
	remoteCmd := fmt.Sprintf("%s \"docker info\"", ssh.GetCommand(cfg))
	if _, err := ssh.ExecuteCommand(log, remoteCmd, "Checking remote Docker installation"); err != nil {
		return fmt.Errorf("remote Docker check failed - please ensure Docker is installed on %s: %v", cfg.Host, err)
	}

	return nil
}

// Build builds the Docker image
func Build(cfg *config.Config, log *logger.Logger) error {
	// Check if Dockerfile exists
	if _, err := os.Stat(cfg.Dockerfile); os.IsNotExist(err) {
		return fmt.Errorf("%s not found", cfg.Dockerfile)
	}

	// Build Docker image with build arguments
	buildCmd := fmt.Sprintf("docker build --platform %s", cfg.Platform)

	// Add build arguments to the command
	for key, value := range cfg.BuildArgs {
		buildCmd += fmt.Sprintf(" --build-arg %s=%s", key, value)
	}

	buildCmd += fmt.Sprintf(" -t %s:%s .", cfg.Image, cfg.Tag)

	_, err := ssh.ExecuteCommand(log, buildCmd, "Building Docker image")
	return err
}

// imageRef returns the full image reference (image:tag)
func imageRef(cfg *config.Config) string {
	return fmt.Sprintf("%s:%s", cfg.Image, cfg.Tag)
}

// getImageSize retrieves and sets the image size in stats
func getImageSize(cfg *config.Config, log *logger.Logger, st *stats.Stats, ref string) {
	sizeCmd := fmt.Sprintf("docker image inspect %s --format='{{.Size}}'", ref)
	sizeResult, err := ssh.ExecuteCommand(log, sizeCmd, "Getting image size")
	if err == nil {
		if size, err := strconv.ParseInt(strings.TrimSpace(sizeResult.Stdout), 10, 64); err == nil {
			st.SetImageSize(size)
		}
	}
}

// getLocalLayers retrieves the layer IDs from the local image
func getLocalLayers(log *logger.Logger, ref string) ([]string, error) {
	localLayersCmd := fmt.Sprintf("docker inspect --format='{{range .RootFS.Layers}}{{.}}\n{{end}}' %s", ref)
	localResult, err := ssh.ExecuteCommand(log, localLayersCmd, "Getting local image layers")
	if err != nil {
		return nil, err
	}
	return parseLayerList(localResult.Stdout), nil
}

// getRemoteLayers retrieves cached layer IDs from the remote host
func getRemoteLayers(cfg *config.Config, log *logger.Logger) map[string]bool {
	remoteBlobsCmd := fmt.Sprintf("%s \"docker images -q '%s' 2>/dev/null | xargs -r docker inspect --format='{{range .RootFS.Layers}}{{.}} {{end}}' 2>/dev/null | tr ' ' '\\n' | grep -v '^$' | sort -u\"",
		ssh.GetCommand(cfg), cfg.Image)
	remoteResult, _ := ssh.ExecuteCommand(log, remoteBlobsCmd, "Getting remote image layers")

	remoteLayers := make(map[string]bool)
	for _, layer := range parseLayerList(remoteResult.Stdout) {
		remoteLayers[layer] = true
	}
	return remoteLayers
}

// calculateLayerDiff compares local and remote layers, returns (newLayers, cachedLayers)
func calculateLayerDiff(localLayers []string, remoteLayers map[string]bool) (newCount, cachedCount int) {
	for _, layer := range localLayers {
		if remoteLayers[layer] {
			cachedCount++
		} else {
			newCount++
		}
	}
	return newCount, cachedCount
}

// Transfer transfers the Docker image to the remote host, only sending changed layers
func Transfer(cfg *config.Config, log *logger.Logger, st *stats.Stats) error {
	ref := imageRef(cfg)

	// Get image size for stats
	getImageSize(cfg, log, st, ref)

	// Get local layers
	localLayers, err := getLocalLayers(log, ref)
	if err != nil || len(localLayers) == 0 {
		return fullTransfer(cfg, log, st)
	}

	// Get cached remote layers
	remoteLayers := getRemoteLayers(cfg, log)

	// Calculate layer diff
	newLayers, cachedLayers := calculateLayerDiff(localLayers, remoteLayers)
	st.SetLayerStats(len(localLayers), cachedLayers, newLayers)

	// If no remote layers or all new, use full transfer
	if len(remoteLayers) == 0 || newLayers == len(localLayers) {
		if len(remoteLayers) == 0 {
			log.Info("First deployment - transferring all layers")
		}
		return fullTransfer(cfg, log, st)
	}

	// Log layer statistics
	cachePercent := float64(cachedLayers) / float64(len(localLayers)) * 100
	log.Info(fmt.Sprintf("Layer cache: %d/%d layers cached (%.0f%%), transferring %d changed layers",
		cachedLayers, len(localLayers), cachePercent, newLayers))

	// Use delta transfer for efficiency
	return deltaTransfer(cfg, log, st, ref, localLayers, remoteLayers)
}

// compressionRatio is the estimated gzip compression ratio for Docker images
const compressionRatio = 0.4

// fullTransfer transfers the entire Docker image
func fullTransfer(cfg *config.Config, log *logger.Logger, st *stats.Stats) error {
	ref := imageRef(cfg)

	// Use pv to measure transfer if available, otherwise fall back to basic transfer
	deployCmd := fmt.Sprintf("docker save %s | gzip | pv -f 2>&1 | %s docker load",
		ref, ssh.GetCommand(cfg))
	result, err := ssh.ExecuteCommand(log, deployCmd, "Transferring Docker image to server")
	if err != nil {
		// Fallback without pv
		deployCmd = fmt.Sprintf("docker save %s | gzip | %s docker load",
			ref, ssh.GetCommand(cfg))
		_, err = ssh.ExecuteCommand(log, deployCmd, "Transferring Docker image to server")
		if err != nil {
			return err
		}
		// Estimate transferred bytes as compressed image size
		if st.ImageSize > 0 {
			st.SetTransferredBytes(int64(float64(st.ImageSize) * compressionRatio))
		}
		return nil
	}

	// Parse pv output for transfer size
	transferred := parsePvOutput(result.Stdout + result.Stderr)
	if transferred > 0 {
		st.SetTransferredBytes(transferred)
	} else if st.ImageSize > 0 {
		// Estimate as compressed size
		st.SetTransferredBytes(int64(float64(st.ImageSize) * compressionRatio))
	}
	return nil
}

// getCachedLayerPrefixes extracts the first 12 characters of cached layer hashes
func getCachedLayerPrefixes(localLayers []string, remoteLayers map[string]bool) []string {
	var prefixes []string
	for _, layer := range localLayers {
		if remoteLayers[layer] {
			hash := strings.TrimPrefix(layer, "sha256:")
			if len(hash) >= 12 {
				prefixes = append(prefixes, hash[:12])
			}
		}
	}
	return prefixes
}

// buildDeltaTransferScript builds the shell script for delta transfer
// The script:
// 1. Exports image to tar
// 2. Extracts to temp dir
// 3. Empties layer.tar files for cached layers (keeps structure for docker load)
// 4. Repacks and sends
// 5. Remote loads the image (docker reuses existing layers by content hash)
func buildDeltaTransferScript(imageRef string, cachedPrefixes []string, sshCmd string) string {
	grepPattern := strings.Join(cachedPrefixes, "|")

	return fmt.Sprintf(`set -e
TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT
docker save %s | tar -xf - -C "$TMPDIR"
cd "$TMPDIR"
for dir in $(ls -d */ 2>/dev/null | grep -E '^(%s)' || true); do
    if [ -f "${dir}layer.tar" ]; then
        : > "${dir}layer.tar"
    fi
done
TRANSFER_SIZE=$(tar -cf - . | wc -c)
echo "TRANSFER_SIZE:$TRANSFER_SIZE"
tar -cf - . | gzip | %s docker load`, imageRef, grepPattern, sshCmd)
}

// parseTransferSize extracts the transfer size from command output
func parseTransferSize(output string) int64 {
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "TRANSFER_SIZE:") {
			if size, err := strconv.ParseInt(strings.TrimPrefix(line, "TRANSFER_SIZE:"), 10, 64); err == nil {
				return size
			}
		}
	}
	return 0
}

// deltaTransfer transfers only new layers by emptying cached layer tarballs
func deltaTransfer(cfg *config.Config, log *logger.Logger, st *stats.Stats, ref string, localLayers []string, remoteLayers map[string]bool) error {
	cachedPrefixes := getCachedLayerPrefixes(localLayers, remoteLayers)
	deltaCmd := buildDeltaTransferScript(ref, cachedPrefixes, ssh.GetCommand(cfg))

	result, err := ssh.ExecuteCommand(log, deltaCmd, "Transferring changed layers only")
	if err != nil {
		log.Info("Delta transfer failed, falling back to full transfer")
		return fullTransfer(cfg, log, st)
	}

	// Parse and set transfer size
	if uncompressedSize := parseTransferSize(result.Stdout); uncompressedSize > 0 {
		st.SetTransferredBytes(int64(float64(uncompressedSize) * compressionRatio))
	}

	return nil
}

// parsePvOutput parses pv command output to extract transferred bytes
func parsePvOutput(output string) int64 {
	// pv outputs something like "10.5MiB" or "1.2GiB"
	// Look for the final size in the output
	output = strings.TrimSpace(output)
	lines := strings.Split(output, "\n")
	
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		// Look for size patterns like "10.5MiB", "1.2GiB", "500KiB"
		return parseSizeString(line)
	}
	return 0
}

// parseSizeString parses a size string like "10.5MiB" to bytes
func parseSizeString(s string) int64 {
	s = strings.TrimSpace(s)
	
	// Extract number and unit
	var numStr string
	var unit string
	for i, c := range s {
		if (c >= '0' && c <= '9') || c == '.' {
			numStr += string(c)
		} else {
			unit = s[i:]
			break
		}
	}
	
	if numStr == "" {
		return 0
	}
	
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0
	}
	
	unit = strings.ToUpper(strings.TrimSpace(unit))
	multiplier := int64(1)
	
	switch {
	case strings.HasPrefix(unit, "K"):
		multiplier = 1024
	case strings.HasPrefix(unit, "M"):
		multiplier = 1024 * 1024
	case strings.HasPrefix(unit, "G"):
		multiplier = 1024 * 1024 * 1024
	case strings.HasPrefix(unit, "T"):
		multiplier = 1024 * 1024 * 1024 * 1024
	}
	
	return int64(num * float64(multiplier))
}

// parseLayerList extracts layer IDs from command output
func parseLayerList(output string) []string {
	var layers []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && strings.HasPrefix(line, "sha256:") {
			layers = append(layers, line)
		}
	}
	return layers
}

// buildContainerConfig builds the docker run arguments from config
func buildContainerConfig(cfg *config.Config) []string {
	args := []string{
		"-d",
		"--name", cfg.ContainerName,
		"--restart", cfg.RestartPolicy,
		"-p", fmt.Sprintf("%s:%s", cfg.HostPort, cfg.ContainerPort),
	}

	// Network
	if cfg.Network != "" {
		args = append(args, "--network", cfg.Network)
	}

	// Resource limits
	if cfg.CPUs != "" {
		args = append(args, "--cpus", cfg.CPUs)
	}
	if cfg.Memory != "" {
		args = append(args, "--memory", cfg.Memory)
	}

	// Volumes
	for _, volume := range cfg.Volumes {
		args = append(args, "-v", volume)
	}

	// Environment file
	if cfg.EnvFile != "" {
		args = append(args, fmt.Sprintf("--env-file ~/%s", cfg.EnvFile))
	}

	// Environment variables
	for key, value := range cfg.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
	}

	// Labels
	for key, value := range cfg.Labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", key, value))
	}

	// Health check
	if cfg.HealthCmd != "" {
		args = append(args, "--health-cmd", fmt.Sprintf("'%s'", cfg.HealthCmd))
	}
	if cfg.HealthInterval != "" {
		args = append(args, "--health-interval", cfg.HealthInterval)
	}
	if cfg.HealthTimeout != "" {
		args = append(args, "--health-timeout", cfg.HealthTimeout)
	}
	if cfg.HealthRetries > 0 {
		args = append(args, "--health-retries", fmt.Sprintf("%d", cfg.HealthRetries))
	}
	if cfg.HealthStart != "" {
		args = append(args, "--health-start-period", cfg.HealthStart)
	}

	// Container user
	if cfg.ContainerUser != "" {
		args = append(args, "--user", cfg.ContainerUser)
	}

	// Working directory
	if cfg.Workdir != "" {
		args = append(args, "--workdir", cfg.Workdir)
	}

	// Hostname
	if cfg.Hostname != "" {
		args = append(args, "--hostname", cfg.Hostname)
	}

	// Extra hosts
	for _, host := range cfg.ExtraHosts {
		args = append(args, "--add-host", host)
	}

	// Security options
	if cfg.Privileged {
		args = append(args, "--privileged")
	}
	if cfg.Init {
		args = append(args, "--init")
	}
	if cfg.ReadOnly {
		args = append(args, "--read-only")
	}

	// Capabilities
	for _, cap := range cfg.CapAdd {
		args = append(args, "--cap-add", cap)
	}
	for _, cap := range cfg.CapDrop {
		args = append(args, "--cap-drop", cap)
	}

	// Tmpfs mounts
	for _, tmpfs := range cfg.Tmpfs {
		args = append(args, "--tmpfs", tmpfs)
	}

	// Logging
	if cfg.LogDriver != "" {
		args = append(args, "--log-driver", cfg.LogDriver)
	}
	for key, value := range cfg.LogOpts {
		args = append(args, "--log-opt", fmt.Sprintf("%s=%s", key, value))
	}

	// Entrypoint override
	if cfg.Entrypoint != "" {
		args = append(args, "--entrypoint", cfg.Entrypoint)
	}

	// Image
	args = append(args, fmt.Sprintf("%s:%s", cfg.Image, cfg.Tag))

	// Command override (must be after image)
	if cfg.Command != "" {
		args = append(args, cfg.Command)
	}

	return args
}

// Deploy deploys the container on the remote host
func Deploy(cfg *config.Config, log *logger.Logger) error {
	containerConfig := buildContainerConfig(cfg)

	remoteCommands := strings.Join([]string{
		fmt.Sprintf("docker stop %s || true", cfg.ContainerName),
		fmt.Sprintf("docker rm %s || true", cfg.ContainerName),
		fmt.Sprintf("docker run %s", strings.Join(containerConfig, " ")),
	}, " && ")

	// Execute remote commands
	restartCmd := fmt.Sprintf("%s \"%s\"", ssh.GetCommand(cfg), remoteCommands)
	if _, err := ssh.ExecuteCommand(log, restartCmd, "Restarting container on server"); err != nil {
		return err
	}

	// Clean up old releases
	if err := cleanupOldReleases(cfg, log); err != nil {
		log.Info(fmt.Sprintf("failed to cleanup old releases: %v", err))
	}

	return verifyContainer(cfg, log)
}

// cleanupOldReleases ensures only the last 5 releases are kept
func cleanupOldReleases(cfg *config.Config, log *logger.Logger) error {
	// Get all images for the current application
	listCmd := fmt.Sprintf("%s \"docker images '%s' --format '{{.Tag}}'\"",
		ssh.GetCommand(cfg), cfg.Image)

	result, err := ssh.ExecuteCommand(log, listCmd, "Listing existing releases")
	if err != nil {
		return err
	}

	// Split tags into slice and reverse the order
	tags := strings.Split(strings.TrimSpace(result.Stdout), "\n")

	if len(tags) <= 5 {
		return nil // No cleanup needed
	}

	slices.Reverse(tags)

	// Remove all but the latest 5 tags
	for _, tag := range tags[5:] {
		if tag == "" {
			continue
		}
		removeCmd := fmt.Sprintf("%s \"docker rmi %s:%s\"",
			ssh.GetCommand(cfg), cfg.Image, tag)

		if _, err := ssh.ExecuteCommand(log, removeCmd,
			fmt.Sprintf("Removing old release %s", tag)); err != nil {
			log.Info(fmt.Sprintf("Failed to remove old release %s: %v", tag, err))
			// Continue with other deletions even if one fails
		}
	}

	return nil
}

// verifyContainer verifies that the container is running
func verifyContainer(cfg *config.Config, log *logger.Logger) error {
	verifyCmd := fmt.Sprintf("%s \"docker ps --filter name=%s --format '{{.Status}}'\"",
		ssh.GetCommand(cfg), cfg.ContainerName)
	result, err := ssh.ExecuteCommand(log, verifyCmd, "Verifying container status")
	if err != nil {
		return err
	}

	if !strings.Contains(result.Stdout, "Up") {
		return fmt.Errorf("container failed to start properly")
	}

	return nil
}

