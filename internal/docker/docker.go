package docker

import (
	"context"
	"fmt"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/bjarneo/pipe/internal/config"
	"github.com/bjarneo/pipe/internal/logger"
	"github.com/bjarneo/pipe/internal/ssh"
	"github.com/bjarneo/pipe/internal/stats"
)

// =============================================================================
// Types
// =============================================================================

// TransferResult contains information about the image transfer
type TransferResult struct {
	NewLayers        int
	CachedLayers     int
	TotalLayers      int
	TransferredBytes int64
	ImageSize        int64
}

// compressionRatio is the estimated gzip compression ratio for Docker images
const compressionRatio = 0.4

// =============================================================================
// Pre-flight Checks
// =============================================================================

// Check verifies Docker is installed and running locally and remotely
func Check(ctx context.Context, cfg *config.Config, log *logger.Logger) error {
	if _, err := ssh.ExecuteCommandContext(ctx, log, "docker info", "Checking local Docker"); err != nil {
		return fmt.Errorf("local Docker check failed: %w", err)
	}

	remoteCmd := fmt.Sprintf("%s \"docker info\"", ssh.GetCommand(cfg))
	if _, err := ssh.ExecuteCommandContext(ctx, log, remoteCmd, "Checking remote Docker"); err != nil {
		return fmt.Errorf("remote Docker check failed - ensure Docker is installed on %s: %w", cfg.Host, err)
	}

	return nil
}

// =============================================================================
// Build
// =============================================================================

// Build builds the Docker image locally
func Build(ctx context.Context, cfg *config.Config, log *logger.Logger) error {
	if _, err := os.Stat(cfg.Dockerfile); os.IsNotExist(err) {
		return fmt.Errorf("%s not found", cfg.Dockerfile)
	}

	cmd := buildDockerBuildCmd(cfg)
	_, err := ssh.ExecuteCommandContext(ctx, log, cmd, "Building Docker image")
	return err
}

// buildDockerBuildCmd constructs the docker build command
func buildDockerBuildCmd(cfg *config.Config) string {
	var parts []string
	parts = append(parts, "docker", "build", "--platform", cfg.Platform)

	// Sort keys for deterministic command generation
	for _, key := range sortedKeys(cfg.BuildArgs) {
		parts = append(parts, "--build-arg", fmt.Sprintf("%s=%s", key, cfg.BuildArgs[key]))
	}

	parts = append(parts, "-t", imageRef(cfg), ".")
	return strings.Join(parts, " ")
}

// sortedKeys returns the keys of a map in sorted order
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// imageRef returns the full image reference (image:tag)
func imageRef(cfg *config.Config) string {
	return fmt.Sprintf("%s:%s", cfg.Image, cfg.Tag)
}

// =============================================================================
// Transfer
// =============================================================================

// Transfer transfers the Docker image to the remote host using delta transfer
func Transfer(ctx context.Context, cfg *config.Config, log *logger.Logger, st *stats.Stats) (*TransferResult, error) {
	ref := imageRef(cfg)
	result := &TransferResult{}

	// Get image size
	if size := getImageSize(ctx, log, ref); size > 0 {
		st.SetImageSize(size)
		result.ImageSize = size
	}

	// Get local layers
	localLayers, err := getLocalLayers(ctx, log, ref)
	if err != nil || len(localLayers) == 0 {
		return doFullTransfer(ctx, cfg, log, st, result, len(localLayers))
	}

	// Get remote layers and calculate diff
	remoteLayers := getRemoteLayers(ctx, cfg, log)
	newLayers, cachedLayers := calculateLayerDiff(localLayers, remoteLayers)

	st.SetLayerStats(len(localLayers), cachedLayers, newLayers)
	result.TotalLayers = len(localLayers)
	result.NewLayers = newLayers
	result.CachedLayers = cachedLayers

	// Choose transfer strategy
	if shouldUseFullTransfer(remoteLayers, newLayers, len(localLayers)) {
		if len(remoteLayers) == 0 {
			log.Info("First deployment - transferring all layers")
		}
		return doFullTransfer(ctx, cfg, log, st, result, len(localLayers))
	}

	// Delta transfer
	logLayerStats(log, cachedLayers, len(localLayers), newLayers)

	if err := deltaTransfer(ctx, cfg, log, st, ref, localLayers, remoteLayers); err != nil {
		return nil, err
	}
	result.TransferredBytes = st.TransferredBytes
	return result, nil
}

// doFullTransfer performs a full image transfer
func doFullTransfer(ctx context.Context, cfg *config.Config, log *logger.Logger, st *stats.Stats, result *TransferResult, layerCount int) (*TransferResult, error) {
	if err := fullTransfer(ctx, cfg, log, st); err != nil {
		return nil, err
	}
	result.TransferredBytes = st.TransferredBytes
	result.TotalLayers = layerCount
	result.NewLayers = layerCount
	return result, nil
}

// shouldUseFullTransfer determines if we should use full transfer instead of delta
func shouldUseFullTransfer(remoteLayers map[string]bool, newLayers, totalLayers int) bool {
	return len(remoteLayers) == 0 || newLayers == totalLayers
}

// logLayerStats logs the layer cache statistics
func logLayerStats(log *logger.Logger, cached, total, new int) {
	cachePercent := float64(cached) / float64(total) * 100
	log.Info(fmt.Sprintf("Layer cache: %d/%d layers cached (%.0f%%), transferring %d changed layers",
		cached, total, cachePercent, new))
}

// getImageSize retrieves the image size in bytes
func getImageSize(ctx context.Context, log *logger.Logger, ref string) int64 {
	cmd := fmt.Sprintf("docker image inspect %s --format='{{.Size}}'", ref)
	result, err := ssh.ExecuteCommandContext(ctx, log, cmd, "Getting image size")
	if err != nil {
		return 0
	}
	size, err := strconv.ParseInt(strings.TrimSpace(result.Stdout), 10, 64)
	if err != nil {
		log.Info(fmt.Sprintf("Failed to parse image size: %v", err))
		return 0
	}
	return size
}

// getLocalLayers retrieves the layer IDs from the local image
func getLocalLayers(ctx context.Context, log *logger.Logger, ref string) ([]string, error) {
	cmd := fmt.Sprintf("docker inspect --format='{{range .RootFS.Layers}}{{.}}\n{{end}}' %s", ref)
	result, err := ssh.ExecuteCommandContext(ctx, log, cmd, "Getting local image layers")
	if err != nil {
		return nil, err
	}
	return parseLayerList(result.Stdout), nil
}

// getRemoteLayers retrieves cached layer IDs from the remote host
func getRemoteLayers(ctx context.Context, cfg *config.Config, log *logger.Logger) map[string]bool {
	cmd := fmt.Sprintf(
		"%s \"docker images -q '%s' 2>/dev/null | xargs -r docker inspect --format='{{range .RootFS.Layers}}{{.}} {{end}}' 2>/dev/null | tr ' ' '\\n' | grep -v '^$' | sort -u\"",
		ssh.GetCommand(cfg), cfg.Image)

	result, err := ssh.ExecuteCommandContext(ctx, log, cmd, "Getting remote image layers")

	layers := make(map[string]bool)
	if err != nil {
		// No remote layers available (first deployment or connection issue)
		return layers
	}
	for _, layer := range parseLayerList(result.Stdout) {
		layers[layer] = true
	}
	return layers
}

// calculateLayerDiff compares local and remote layers
func calculateLayerDiff(localLayers []string, remoteLayers map[string]bool) (newCount, cachedCount int) {
	for _, layer := range localLayers {
		if remoteLayers[layer] {
			cachedCount++
		} else {
			newCount++
		}
	}
	return
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

// =============================================================================
// Full Transfer
// =============================================================================

// fullTransfer transfers the entire Docker image
func fullTransfer(ctx context.Context, cfg *config.Config, log *logger.Logger, st *stats.Stats) error {
	ref := imageRef(cfg)
	sshCmd := ssh.GetCommand(cfg)

	// Try with pv for progress monitoring
	cmd := fmt.Sprintf("docker save %s | gzip | pv -f 2>&1 | %s docker load", ref, sshCmd)
	result, err := ssh.ExecuteCommandContext(ctx, log, cmd, "Transferring Docker image")

	if err != nil {
		// Fallback without pv
		cmd = fmt.Sprintf("docker save %s | gzip | %s docker load", ref, sshCmd)
		if _, err := ssh.ExecuteCommandContext(ctx, log, cmd, "Transferring Docker image"); err != nil {
			return err
		}
		estimateTransferSize(st)
		return nil
	}

	parseAndSetTransferSize(st, result.Stdout+result.Stderr)
	return nil
}

// estimateTransferSize estimates transfer size based on image size
func estimateTransferSize(st *stats.Stats) {
	if st.ImageSize > 0 {
		st.SetTransferredBytes(int64(float64(st.ImageSize) * compressionRatio))
	}
}

// parseAndSetTransferSize parses pv output and sets transfer size
func parseAndSetTransferSize(st *stats.Stats, output string) {
	if transferred := parsePvOutput(output); transferred > 0 {
		st.SetTransferredBytes(transferred)
	} else {
		estimateTransferSize(st)
	}
}

// =============================================================================
// Delta Transfer
// =============================================================================

// deltaTransfer transfers only new layers by emptying cached layer tarballs
func deltaTransfer(ctx context.Context, cfg *config.Config, log *logger.Logger, st *stats.Stats, ref string, localLayers []string, remoteLayers map[string]bool) error {
	cachedPrefixes := getCachedLayerPrefixes(localLayers, remoteLayers)
	script := buildDeltaTransferScript(ref, cachedPrefixes, ssh.GetCommand(cfg))

	result, err := ssh.ExecuteCommandContext(ctx, log, script, "Transferring changed layers only")
	if err != nil {
		log.Info("Delta transfer failed, falling back to full transfer")
		return fullTransfer(ctx, cfg, log, st)
	}

	if size := parseTransferSize(result.Stdout); size > 0 {
		st.SetTransferredBytes(int64(float64(size) * compressionRatio))
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

// =============================================================================
// Size Parsing
// =============================================================================

// parsePvOutput parses pv command output to extract transferred bytes
func parsePvOutput(output string) int64 {
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			return parseSizeString(line)
		}
	}
	return 0
}

// parseSizeString parses a size string like "10.5MiB" to bytes
func parseSizeString(s string) int64 {
	s = strings.TrimSpace(s)

	var numStr, unit string
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

	return int64(num * float64(sizeMultiplier(unit)))
}

// sizeMultiplier returns the byte multiplier for a size unit
func sizeMultiplier(unit string) int64 {
	switch strings.ToUpper(strings.TrimSpace(unit)) {
	case "K", "KB", "KIB":
		return 1024
	case "M", "MB", "MIB":
		return 1024 * 1024
	case "G", "GB", "GIB":
		return 1024 * 1024 * 1024
	case "T", "TB", "TIB":
		return 1024 * 1024 * 1024 * 1024
	default:
		return 1
	}
}

// =============================================================================
// Deploy
// =============================================================================

// Deploy deploys the container on the remote host
func Deploy(ctx context.Context, cfg *config.Config, log *logger.Logger) error {
	containerArgs := NewRunBuilder(cfg).Build()

	commands := []string{
		fmt.Sprintf("docker stop %s || true", cfg.ContainerName),
		fmt.Sprintf("docker rm %s || true", cfg.ContainerName),
		fmt.Sprintf("docker run %s", strings.Join(containerArgs, " ")),
	}

	cmd := fmt.Sprintf("%s \"%s\"", ssh.GetCommand(cfg), strings.Join(commands, " && "))
	if _, err := ssh.ExecuteCommandContext(ctx, log, cmd, "Starting container"); err != nil {
		return err
	}

	if err := cleanupOldReleases(ctx, cfg, log); err != nil {
		log.Info(fmt.Sprintf("failed to cleanup old releases: %v", err))
	}

	return verifyContainer(ctx, cfg, log)
}

// cleanupOldReleases ensures only the last 5 releases are kept
func cleanupOldReleases(ctx context.Context, cfg *config.Config, log *logger.Logger) error {
	cmd := fmt.Sprintf("%s \"docker images '%s' --format '{{.Tag}}'\"",
		ssh.GetCommand(cfg), cfg.Image)

	result, err := ssh.ExecuteCommandContext(ctx, log, cmd, "Listing existing releases")
	if err != nil {
		return err
	}

	tags := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	if len(tags) <= 5 {
		return nil
	}

	slices.Reverse(tags)
	for _, tag := range tags[5:] {
		if tag == "" {
			continue
		}
		removeCmd := fmt.Sprintf("%s \"docker rmi %s:%s\"", ssh.GetCommand(cfg), cfg.Image, tag)
		if _, err := ssh.ExecuteCommandContext(ctx, log, removeCmd, fmt.Sprintf("Removing old release %s", tag)); err != nil {
			log.Info(fmt.Sprintf("Failed to remove old release %s: %v", tag, err))
		}
	}

	return nil
}

// verifyContainer verifies that the container is running
func verifyContainer(ctx context.Context, cfg *config.Config, log *logger.Logger) error {
	cmd := fmt.Sprintf("%s \"docker ps --filter name=%s --format '{{.Status}}'\"",
		ssh.GetCommand(cfg), cfg.ContainerName)

	result, err := ssh.ExecuteCommandContext(ctx, log, cmd, "Verifying container status")
	if err != nil {
		return err
	}

	if !strings.Contains(result.Stdout, "Up") {
		return fmt.Errorf("container failed to start properly")
	}

	return nil
}

// =============================================================================
// Container Run Builder
// =============================================================================

// RunBuilder constructs docker run arguments using a fluent interface
type RunBuilder struct {
	args []string
	cfg  *config.Config
}

// NewRunBuilder creates a new RunBuilder from config
func NewRunBuilder(cfg *config.Config) *RunBuilder {
	return &RunBuilder{cfg: cfg}
}

// Build constructs and returns all docker run arguments
func (b *RunBuilder) Build() []string {
	return b.BuildWithImage(imageRef(b.cfg))
}

// BuildWithImage constructs docker run arguments with a custom image
// This is useful for rollback operations where we need to use a previous image
func (b *RunBuilder) BuildWithImage(image string) []string {
	b.addCore()
	b.addNetwork()
	b.addResources()
	b.addVolumes()
	b.addEnvironment()
	b.addLabels()
	b.addHealthCheck()
	b.addUserConfig()
	b.addSecurity()
	b.addStorage()
	b.addLogging()
	b.addOverrides()
	b.addImageWithRef(image)
	return b.args
}

// addCore adds core container options
func (b *RunBuilder) addCore() {
	b.args = append(b.args,
		"-d",
		"--name", b.cfg.ContainerName,
		"--restart", b.cfg.RestartPolicy,
		"-p", fmt.Sprintf("%s:%s", b.cfg.HostPort, b.cfg.ContainerPort),
	)
}

// addNetwork adds network configuration
func (b *RunBuilder) addNetwork() {
	if b.cfg.Network != "" {
		b.args = append(b.args, "--network", b.cfg.Network)
	}
}

// addResources adds resource limits
func (b *RunBuilder) addResources() {
	if b.cfg.CPUs != "" {
		b.args = append(b.args, "--cpus", b.cfg.CPUs)
	}
	if b.cfg.Memory != "" {
		b.args = append(b.args, "--memory", b.cfg.Memory)
	}
}

// addVolumes adds volume mounts
func (b *RunBuilder) addVolumes() {
	for _, vol := range b.cfg.Volumes {
		b.args = append(b.args, "-v", vol)
	}
}

// addEnvironment adds environment configuration
func (b *RunBuilder) addEnvironment() {
	if b.cfg.EnvFile != "" {
		b.args = append(b.args, fmt.Sprintf("--env-file ~/%s", b.cfg.EnvFile))
	}
	// Sort keys for deterministic command generation
	for _, key := range sortedKeys(b.cfg.Env) {
		b.args = append(b.args, "-e", fmt.Sprintf("%s=%s", key, b.cfg.Env[key]))
	}
}

// addLabels adds container labels
func (b *RunBuilder) addLabels() {
	// Sort keys for deterministic command generation
	for _, key := range sortedKeys(b.cfg.Labels) {
		b.args = append(b.args, "--label", fmt.Sprintf("%s=%s", key, b.cfg.Labels[key]))
	}
}

// addHealthCheck adds health check configuration
func (b *RunBuilder) addHealthCheck() {
	if b.cfg.HealthCmd != "" {
		b.args = append(b.args, "--health-cmd", fmt.Sprintf("'%s'", b.cfg.HealthCmd))
	}
	if b.cfg.HealthInterval != "" {
		b.args = append(b.args, "--health-interval", b.cfg.HealthInterval)
	}
	if b.cfg.HealthTimeout != "" {
		b.args = append(b.args, "--health-timeout", b.cfg.HealthTimeout)
	}
	if b.cfg.HealthRetries > 0 {
		b.args = append(b.args, "--health-retries", fmt.Sprintf("%d", b.cfg.HealthRetries))
	}
	if b.cfg.HealthStart != "" {
		b.args = append(b.args, "--health-start-period", b.cfg.HealthStart)
	}
}

// addUserConfig adds user/workdir/hostname configuration
func (b *RunBuilder) addUserConfig() {
	if b.cfg.ContainerUser != "" {
		b.args = append(b.args, "--user", b.cfg.ContainerUser)
	}
	if b.cfg.Workdir != "" {
		b.args = append(b.args, "--workdir", b.cfg.Workdir)
	}
	if b.cfg.Hostname != "" {
		b.args = append(b.args, "--hostname", b.cfg.Hostname)
	}
	for _, host := range b.cfg.ExtraHosts {
		b.args = append(b.args, "--add-host", host)
	}
}

// addSecurity adds security options
func (b *RunBuilder) addSecurity() {
	if b.cfg.Privileged {
		b.args = append(b.args, "--privileged")
	}
	if b.cfg.Init {
		b.args = append(b.args, "--init")
	}
	if b.cfg.ReadOnly {
		b.args = append(b.args, "--read-only")
	}
	for _, cap := range b.cfg.CapAdd {
		b.args = append(b.args, "--cap-add", cap)
	}
	for _, cap := range b.cfg.CapDrop {
		b.args = append(b.args, "--cap-drop", cap)
	}
}

// addStorage adds tmpfs mounts
func (b *RunBuilder) addStorage() {
	for _, tmpfs := range b.cfg.Tmpfs {
		b.args = append(b.args, "--tmpfs", tmpfs)
	}
}

// addLogging adds logging configuration
func (b *RunBuilder) addLogging() {
	if b.cfg.LogDriver != "" {
		b.args = append(b.args, "--log-driver", b.cfg.LogDriver)
	}
	// Sort keys for deterministic command generation
	for _, key := range sortedKeys(b.cfg.LogOpts) {
		b.args = append(b.args, "--log-opt", fmt.Sprintf("%s=%s", key, b.cfg.LogOpts[key]))
	}
}

// addOverrides adds entrypoint and command overrides
func (b *RunBuilder) addOverrides() {
	if b.cfg.Entrypoint != "" {
		b.args = append(b.args, "--entrypoint", b.cfg.Entrypoint)
	}
}

// addImage adds the image reference and optional command
func (b *RunBuilder) addImage() {
	b.addImageWithRef(imageRef(b.cfg))
}

// addImageWithRef adds a custom image reference and optional command
func (b *RunBuilder) addImageWithRef(image string) {
	b.args = append(b.args, image)
	if b.cfg.Command != "" {
		b.args = append(b.args, b.cfg.Command)
	}
}
