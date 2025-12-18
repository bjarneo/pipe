package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the deployment configuration
type Config struct {
	Host           string            `json:"host" yaml:"host"`
	User           string            `json:"user" yaml:"user"`
	Image          string            `json:"image" yaml:"image"`
	Dockerfile     string            `json:"dockerfile" yaml:"dockerfile"`
	Tag            string            `json:"tag" yaml:"tag"`
	Platform       string            `json:"platform" yaml:"platform"`
	SSHKey         string            `json:"sshKey" yaml:"sshKey"`
	SSHPort        string            `json:"sshPort" yaml:"sshPort"`
	ContainerName  string            `json:"containerName" yaml:"containerName"`
	ContainerPort  string            `json:"containerPort" yaml:"containerPort"`
	HostPort       string            `json:"hostPort" yaml:"hostPort"`
	EnvFile        string            `json:"envFile" yaml:"envFile"`
	Rollback       bool              `json:"rollback" yaml:"rollback"`
	BuildArgs      map[string]string `json:"buildArgs" yaml:"buildArgs"`
	Network        string            `json:"network" yaml:"network"`
	Volumes        []string          `json:"volumes" yaml:"volumes"`
	CPUs           string            `json:"cpus" yaml:"cpus"`
	Memory         string            `json:"memory" yaml:"memory"`
	RemoteCommands []string          `json:"remoteCommands" yaml:"remoteCommands"`

	// Health check options
	HealthCmd      string `json:"healthCmd" yaml:"healthCmd"`
	HealthInterval string `json:"healthInterval" yaml:"healthInterval"`
	HealthTimeout  string `json:"healthTimeout" yaml:"healthTimeout"`
	HealthRetries  int    `json:"healthRetries" yaml:"healthRetries"`
	HealthStart    string `json:"healthStartPeriod" yaml:"healthStartPeriod"`

	// Container runtime options
	RestartPolicy string            `json:"restartPolicy" yaml:"restartPolicy"`
	Labels        map[string]string `json:"labels" yaml:"labels"`
	Env           map[string]string `json:"env" yaml:"env"`
	Entrypoint    string            `json:"entrypoint" yaml:"entrypoint"`
	Command       string            `json:"command" yaml:"command"`
	ContainerUser string            `json:"containerUser" yaml:"containerUser"`
	Workdir       string            `json:"workdir" yaml:"workdir"`
	Hostname      string            `json:"hostname" yaml:"hostname"`
	ExtraHosts    []string          `json:"extraHosts" yaml:"extraHosts"`
	Privileged    bool              `json:"privileged" yaml:"privileged"`
	Init          bool              `json:"init" yaml:"init"`
	ReadOnly      bool              `json:"readOnly" yaml:"readOnly"`
	CapAdd        []string          `json:"capAdd" yaml:"capAdd"`
	CapDrop       []string          `json:"capDrop" yaml:"capDrop"`
	Tmpfs         []string          `json:"tmpfs" yaml:"tmpfs"`
	LogDriver     string            `json:"logDriver" yaml:"logDriver"`
	LogOpts       map[string]string `json:"logOpts" yaml:"logOpts"`

	// Execution options
	DryRun  bool `json:"dryRun" yaml:"dryRun"`
	Verbose bool `json:"verbose" yaml:"verbose"`
}

// arrayFlags allows for multiple flag values
type arrayFlags []string

func (i *arrayFlags) String() string {
	return strings.Join(*i, ",")
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

// Load loads configuration from command line flags and environment variables
func Load() Config {
	var config Config
	var showHelp bool
	var showVersion bool
	var buildArgs arrayFlags
	var volumeFlags arrayFlags
	var remoteCommandFlags arrayFlags
	var envFlags arrayFlags
	var labelFlags arrayFlags
	var extraHostFlags arrayFlags
	var capAddFlags arrayFlags
	var capDropFlags arrayFlags
	var tmpfsFlags arrayFlags
	var logOptFlags arrayFlags
	var configFile string

	// Initialize maps
	config.BuildArgs = make(map[string]string)
	config.Labels = make(map[string]string)
	config.Env = make(map[string]string)
	config.LogOpts = make(map[string]string)

	// Define command line flags
	flag.StringVar(&configFile, "config", "", "Path to config file (default: pipe.yaml or pipe.yml)")
	flag.StringVar(&config.Host, "host", "", "Remote host to deploy to")
	flag.StringVar(&config.User, "user", "", "SSH user for remote host")
	flag.StringVar(&config.SSHPort, "ssh-port", "", "SSH port (default: 22)")
	flag.StringVar(&config.Image, "image", "", "Docker image name")
	flag.StringVar(&config.Dockerfile, "dockerfile", "", "Path to the Dockerfile")
	flag.StringVar(&config.Tag, "tag", "", "Docker image tag")
	flag.StringVar(&config.Platform, "platform", "", "Docker platform")
	flag.StringVar(&config.SSHKey, "ssh-key", "", "Path to SSH key")
	flag.StringVar(&config.ContainerName, "container-name", "", "Name for the container")
	flag.StringVar(&config.ContainerPort, "container-port", "", "Container port")
	flag.StringVar(&config.HostPort, "host-port", "", "Host port")
	flag.StringVar(&config.EnvFile, "env-file", "", "Environment file")
	flag.Var(&buildArgs, "build-arg", "Build argument in KEY=VALUE format (can be specified multiple times)")
	flag.Var(&volumeFlags, "volume", "Volume mount in format 'host:container' (can be specified multiple times)")
	flag.Var(&remoteCommandFlags, "remote-command", "Remote command to execute after deployment (can be specified multiple times)")
	flag.StringVar(&config.Network, "network", "", "Docker network to connect to")
	flag.StringVar(&config.CPUs, "cpus", "", "Number of CPUs (e.g., '0.5' or '2')")
	flag.StringVar(&config.Memory, "memory", "", "Memory limit (e.g., '512m' or '2g')")

	// Health check flags
	flag.StringVar(&config.HealthCmd, "health-cmd", "", "Health check command")
	flag.StringVar(&config.HealthInterval, "health-interval", "", "Health check interval (e.g., '30s')")
	flag.StringVar(&config.HealthTimeout, "health-timeout", "", "Health check timeout (e.g., '10s')")
	flag.IntVar(&config.HealthRetries, "health-retries", 0, "Health check retries")
	flag.StringVar(&config.HealthStart, "health-start-period", "", "Health check start period (e.g., '5s')")

	// Container runtime flags
	flag.StringVar(&config.RestartPolicy, "restart", "", "Restart policy (no, always, on-failure, unless-stopped)")
	flag.Var(&labelFlags, "label", "Container label in KEY=VALUE format (can be specified multiple times)")
	flag.Var(&envFlags, "env", "Environment variable in KEY=VALUE format (can be specified multiple times)")
	flag.StringVar(&config.Entrypoint, "entrypoint", "", "Override container entrypoint")
	flag.StringVar(&config.Command, "command", "", "Override container command")
	flag.StringVar(&config.ContainerUser, "container-user", "", "User to run container as (user or user:group)")
	flag.StringVar(&config.Workdir, "workdir", "", "Working directory inside container")
	flag.StringVar(&config.Hostname, "hostname", "", "Container hostname")
	flag.Var(&extraHostFlags, "add-host", "Add host-to-IP mapping (host:ip)")
	flag.BoolVar(&config.Privileged, "privileged", false, "Run container in privileged mode")
	flag.BoolVar(&config.Init, "init", false, "Run init inside container")
	flag.BoolVar(&config.ReadOnly, "read-only", false, "Mount root filesystem as read-only")
	flag.Var(&capAddFlags, "cap-add", "Add Linux capability")
	flag.Var(&capDropFlags, "cap-drop", "Drop Linux capability")
	flag.Var(&tmpfsFlags, "tmpfs", "Mount tmpfs (path or path:opts)")
	flag.StringVar(&config.LogDriver, "log-driver", "", "Logging driver (e.g., json-file, syslog, none)")
	flag.Var(&logOptFlags, "log-opt", "Log driver options in KEY=VALUE format")

	// Execution flags
	flag.BoolVar(&config.DryRun, "dry-run", false, "Preview deployment without executing")
	flag.BoolVar(&config.Verbose, "verbose", false, "Show detailed output")
	flag.BoolVar(&config.Verbose, "v", false, "Show detailed output (shorthand)")
	flag.BoolVar(&showHelp, "help", false, "Show help message")
	flag.BoolVar(&config.Rollback, "rollback", false, "Rollback to previous version")
	flag.BoolVar(&showVersion, "version", false, "Show version information")

	// Custom usage message
	flag.Usage = func() {
		fmt.Print(helpText)
	}

	// Parse command line flags
	flag.Parse()

	// Show help if requested
	if showHelp {
		flag.Usage()
		os.Exit(0)
	}

	if showVersion {
		fmt.Printf("pipe version %s\n", version)
		os.Exit(0)
	}

	// Load config file first (lowest priority)
	fileConfig := loadConfigFile(configFile)

	// Merge: config file -> env vars -> CLI flags (CLI has highest priority)
	config = mergeConfig(fileConfig, config, buildArgs, volumeFlags, remoteCommandFlags,
		envFlags, labelFlags, extraHostFlags, capAddFlags, capDropFlags, tmpfsFlags, logOptFlags)

	// Expand home directory in SSH key path
	if strings.HasPrefix(config.SSHKey, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to expand ~ in SSH key path: %v\n", err)
			fmt.Fprintf(os.Stderr, "SSH key path '%s' may not work correctly\n", config.SSHKey)
		} else {
			config.SSHKey = filepath.Join(home, config.SSHKey[2:])
		}
	}

	return config
}

// Validation patterns - these prevent shell injection by ensuring no dangerous characters
var (
	// Hostname: alphanumeric, dots, hyphens (RFC 1123)
	hostnameRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-\.]*[a-zA-Z0-9])?$`)
	// IP address pattern
	ipRegex = regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
	// Unix username: starts with letter or underscore, alphanumeric/underscore/hyphen
	usernameRegex = regexp.MustCompile(`^[a-z_][a-z0-9_-]*$`)
	// Docker image name: lowercase alphanumeric, can include slashes, dots, hyphens, underscores
	imageNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9._/-]*$`)
	// Docker tag: alphanumeric, dots, hyphens, underscores
	tagRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)
	// Container name: alphanumeric, underscores, hyphens
	containerNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)
	// Network name: alphanumeric, underscores, hyphens
	networkNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)
	// Build arg key: uppercase alphanumeric and underscores
	buildArgKeyRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	// CPU format: decimal number
	cpuRegex = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?$`)
	// Memory format: number followed by optional unit
	memoryRegex = regexp.MustCompile(`^[0-9]+[bkmgBKMG]?$`)
	// Dangerous shell characters that should never appear in unquoted input
	dangerousCharsRegex = regexp.MustCompile(`[;&|$` + "`" + `\\\n\r"'<>(){}]`)
)

// Valid platforms whitelist
var validPlatforms = map[string]bool{
	"linux/amd64":   true,
	"linux/arm64":   true,
	"linux/arm/v7":  true,
	"linux/arm/v6":  true,
	"linux/386":     true,
	"linux/ppc64le": true,
	"linux/s390x":   true,
}

// Validate validates the configuration and returns an error if any field is invalid
// This is critical for preventing command injection attacks
func (c *Config) Validate() error {
	var errors []string

	// Required fields
	if c.Host == "" {
		errors = append(errors, "host is required")
	} else if !hostnameRegex.MatchString(c.Host) && !ipRegex.MatchString(c.Host) {
		errors = append(errors, "host must be a valid hostname or IP address")
	}

	if c.User == "" {
		errors = append(errors, "user is required")
	} else if !usernameRegex.MatchString(c.User) {
		errors = append(errors, "user must be a valid Unix username (lowercase, starts with letter/underscore)")
	}

	// Image validation
	if c.Image != "" && !imageNameRegex.MatchString(c.Image) {
		errors = append(errors, "image name contains invalid characters")
	}

	// Tag validation
	if c.Tag != "" && !tagRegex.MatchString(c.Tag) {
		errors = append(errors, "tag contains invalid characters")
	}

	// Container name validation
	if c.ContainerName != "" && !containerNameRegex.MatchString(c.ContainerName) {
		errors = append(errors, "container-name contains invalid characters")
	}

	// Platform validation (whitelist)
	if c.Platform != "" && !validPlatforms[c.Platform] {
		errors = append(errors, fmt.Sprintf("platform must be one of: linux/amd64, linux/arm64, linux/arm/v7, linux/arm/v6, linux/386"))
	}

	// Port validation
	if c.ContainerPort != "" {
		if port, err := strconv.Atoi(c.ContainerPort); err != nil || port < 1 || port > 65535 {
			errors = append(errors, "container-port must be a valid port number (1-65535)")
		}
	}

	if c.HostPort != "" {
		if port, err := strconv.Atoi(c.HostPort); err != nil || port < 1 || port > 65535 {
			errors = append(errors, "host-port must be a valid port number (1-65535)")
		}
	}

	// Network validation
	if c.Network != "" && !networkNameRegex.MatchString(c.Network) {
		errors = append(errors, "network name contains invalid characters")
	}

	// CPU validation
	if c.CPUs != "" && !cpuRegex.MatchString(c.CPUs) {
		errors = append(errors, "cpus must be a valid number (e.g., '0.5' or '2')")
	}

	// Memory validation
	if c.Memory != "" && !memoryRegex.MatchString(c.Memory) {
		errors = append(errors, "memory must be a valid format (e.g., '512m' or '2g')")
	}

	// Build args validation - check for shell injection
	for key, value := range c.BuildArgs {
		if !buildArgKeyRegex.MatchString(key) {
			errors = append(errors, fmt.Sprintf("build-arg key '%s' contains invalid characters", key))
		}
		if dangerousCharsRegex.MatchString(value) {
			errors = append(errors, fmt.Sprintf("build-arg value for '%s' contains dangerous shell characters", key))
		}
	}

	// Volume validation
	for _, vol := range c.Volumes {
		if vol == "" {
			continue
		}
		if !strings.Contains(vol, ":") {
			errors = append(errors, fmt.Sprintf("volume '%s' must be in format 'host:container'", vol))
		}
		if dangerousCharsRegex.MatchString(vol) {
			errors = append(errors, fmt.Sprintf("volume '%s' contains dangerous shell characters", vol))
		}
		// Check for path traversal
		if strings.Contains(vol, "..") {
			errors = append(errors, fmt.Sprintf("volume '%s' contains path traversal sequence", vol))
		}
	}

	// Env file validation - check for path traversal
	if c.EnvFile != "" {
		if strings.Contains(c.EnvFile, "..") {
			errors = append(errors, "env-file path contains path traversal sequence")
		}
		if filepath.IsAbs(c.EnvFile) {
			errors = append(errors, "env-file must be a relative path")
		}
		if dangerousCharsRegex.MatchString(c.EnvFile) {
			errors = append(errors, "env-file path contains dangerous shell characters")
		}
	}

	// Dockerfile validation - check for path traversal
	if c.Dockerfile != "" {
		if strings.Contains(c.Dockerfile, "..") {
			errors = append(errors, "dockerfile path contains path traversal sequence")
		}
		if dangerousCharsRegex.MatchString(c.Dockerfile) {
			errors = append(errors, "dockerfile path contains dangerous shell characters")
		}
	}

	// SSH key path validation
	if c.SSHKey != "" {
		if dangerousCharsRegex.MatchString(c.SSHKey) {
			errors = append(errors, "ssh-key path contains dangerous shell characters")
		}
	}

	// Remote commands validation
	for i, cmd := range c.RemoteCommands {
		if cmd == "" {
			continue
		}
		// Check for extremely dangerous patterns that could compromise the host
		dangerousPatterns := []string{"rm -rf /", "mkfs", "dd if=", "> /dev/"}
		for _, pattern := range dangerousPatterns {
			if strings.Contains(cmd, pattern) {
				errors = append(errors, fmt.Sprintf("remote-command[%d] contains dangerous pattern '%s'", i, pattern))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// loadConfigFile loads configuration from a YAML file
func loadConfigFile(configPath string) Config {
	var config Config
	config.BuildArgs = make(map[string]string)
	config.Labels = make(map[string]string)
	config.Env = make(map[string]string)
	config.LogOpts = make(map[string]string)

	// Determine which config file to use
	if configPath == "" {
		// Try default config file names
		for _, name := range []string{"pipe.yaml", "pipe.yml"} {
			if _, err := os.Stat(name); err == nil {
				configPath = name
				break
			}
		}
	}

	if configPath == "" {
		return config
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not read config file %s: %v\n", configPath, err)
		return config
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not parse config file %s: %v\n", configPath, err)
		return config
	}

	// Expand environment variables in config values
	config = expandEnvVars(config)

	return config
}

// expandEnvVars expands environment variables in config string values
func expandEnvVars(config Config) Config {
	expand := func(s string) string {
		return os.Expand(s, func(key string) string {
			if val, exists := os.LookupEnv(key); exists {
				return val
			}
			return "${" + key + "}"
		})
	}

	config.Host = expand(config.Host)
	config.User = expand(config.User)
	config.Image = expand(config.Image)
	config.Dockerfile = expand(config.Dockerfile)
	config.Tag = expand(config.Tag)
	config.Platform = expand(config.Platform)
	config.SSHKey = expand(config.SSHKey)
	config.SSHPort = expand(config.SSHPort)
	config.ContainerName = expand(config.ContainerName)
	config.ContainerPort = expand(config.ContainerPort)
	config.HostPort = expand(config.HostPort)
	config.EnvFile = expand(config.EnvFile)
	config.Network = expand(config.Network)
	config.CPUs = expand(config.CPUs)
	config.Memory = expand(config.Memory)

	// Health check
	config.HealthCmd = expand(config.HealthCmd)
	config.HealthInterval = expand(config.HealthInterval)
	config.HealthTimeout = expand(config.HealthTimeout)
	config.HealthStart = expand(config.HealthStart)

	// Container runtime
	config.RestartPolicy = expand(config.RestartPolicy)
	config.Entrypoint = expand(config.Entrypoint)
	config.Command = expand(config.Command)
	config.ContainerUser = expand(config.ContainerUser)
	config.Workdir = expand(config.Workdir)
	config.Hostname = expand(config.Hostname)
	config.LogDriver = expand(config.LogDriver)

	// Expand build args
	for key, value := range config.BuildArgs {
		config.BuildArgs[key] = expand(value)
	}

	// Expand labels
	for key, value := range config.Labels {
		config.Labels[key] = expand(value)
	}

	// Expand env
	for key, value := range config.Env {
		config.Env[key] = expand(value)
	}

	// Expand log opts
	for key, value := range config.LogOpts {
		config.LogOpts[key] = expand(value)
	}

	// Expand volumes
	for i, vol := range config.Volumes {
		config.Volumes[i] = expand(vol)
	}

	// Expand remote commands
	for i, cmd := range config.RemoteCommands {
		config.RemoteCommands[i] = expand(cmd)
	}

	// Expand extra hosts
	for i, host := range config.ExtraHosts {
		config.ExtraHosts[i] = expand(host)
	}

	// Expand tmpfs
	for i, t := range config.Tmpfs {
		config.Tmpfs[i] = expand(t)
	}

	return config
}

// mergeConfig merges configuration from file, environment variables, and CLI flags
// Priority: CLI flags > env vars > config file > defaults
func mergeConfig(fileConfig, cliConfig Config, buildArgs, volumeFlags, remoteCommandFlags,
	envFlags, labelFlags, extraHostFlags, capAddFlags, capDropFlags, tmpfsFlags, logOptFlags arrayFlags) Config {
	result := fileConfig

	// Helper to get value with priority: CLI > env > file > default
	getString := func(cliVal, envKey, fileVal, defaultVal string) string {
		if cliVal != "" {
			return cliVal
		}
		if envKey != "" {
			if envVal := os.Getenv(envKey); envVal != "" {
				return envVal
			}
		}
		if fileVal != "" {
			return fileVal
		}
		return defaultVal
	}

	getInt := func(cliVal int, envKey string, fileVal int, defaultVal int) int {
		if cliVal != 0 {
			return cliVal
		}
		if envKey != "" {
			if envVal := os.Getenv(envKey); envVal != "" {
				if v, err := strconv.Atoi(envVal); err == nil {
					return v
				}
			}
		}
		if fileVal != 0 {
			return fileVal
		}
		return defaultVal
	}

	getBool := func(cliVal bool, envKey string, fileVal bool) bool {
		if cliVal {
			return true
		}
		if envKey != "" {
			if envVal := os.Getenv(envKey); envVal != "" {
				return envVal == "true" || envVal == "1" || envVal == "yes"
			}
		}
		return fileVal
	}

	result.Host = getString(cliConfig.Host, "HOST", fileConfig.Host, "")
	result.User = getString(cliConfig.User, "HOST_USER", fileConfig.User, "")
	result.SSHPort = getString(cliConfig.SSHPort, "SSH_PORT", fileConfig.SSHPort, "22")
	result.Image = getString(cliConfig.Image, "DOCKER_IMAGE_NAME", fileConfig.Image, "app")
	result.Dockerfile = getString(cliConfig.Dockerfile, "DOCKERFILE", fileConfig.Dockerfile, "Dockerfile")
	result.Tag = getString(cliConfig.Tag, "DOCKER_IMAGE_TAG", fileConfig.Tag, "latest")
	result.Platform = getString(cliConfig.Platform, "HOST_PLATFORM", fileConfig.Platform, "linux/amd64")
	result.SSHKey = getString(cliConfig.SSHKey, "SSH_KEY_PATH", fileConfig.SSHKey, "")
	result.ContainerName = getString(cliConfig.ContainerName, "DOCKER_CONTAINER_NAME", fileConfig.ContainerName, "app")
	result.ContainerPort = getString(cliConfig.ContainerPort, "DOCKER_CONTAINER_PORT", fileConfig.ContainerPort, "3000")
	result.HostPort = getString(cliConfig.HostPort, "HOST_PORT", fileConfig.HostPort, "3000")
	result.EnvFile = getString(cliConfig.EnvFile, "DOCKER_CONTAINER_ENV_FILE", fileConfig.EnvFile, "")
	result.Network = getString(cliConfig.Network, "DOCKER_NETWORK", fileConfig.Network, "")
	result.CPUs = getString(cliConfig.CPUs, "DOCKER_CPUS", fileConfig.CPUs, "")
	result.Memory = getString(cliConfig.Memory, "DOCKER_MEMORY", fileConfig.Memory, "")

	// Health check options
	result.HealthCmd = getString(cliConfig.HealthCmd, "HEALTH_CMD", fileConfig.HealthCmd, "")
	result.HealthInterval = getString(cliConfig.HealthInterval, "HEALTH_INTERVAL", fileConfig.HealthInterval, "")
	result.HealthTimeout = getString(cliConfig.HealthTimeout, "HEALTH_TIMEOUT", fileConfig.HealthTimeout, "")
	result.HealthRetries = getInt(cliConfig.HealthRetries, "HEALTH_RETRIES", fileConfig.HealthRetries, 0)
	result.HealthStart = getString(cliConfig.HealthStart, "HEALTH_START_PERIOD", fileConfig.HealthStart, "")

	// Container runtime options
	result.RestartPolicy = getString(cliConfig.RestartPolicy, "RESTART_POLICY", fileConfig.RestartPolicy, "unless-stopped")
	result.Entrypoint = getString(cliConfig.Entrypoint, "", fileConfig.Entrypoint, "")
	result.Command = getString(cliConfig.Command, "", fileConfig.Command, "")
	result.ContainerUser = getString(cliConfig.ContainerUser, "CONTAINER_USER", fileConfig.ContainerUser, "")
	result.Workdir = getString(cliConfig.Workdir, "WORKDIR", fileConfig.Workdir, "")
	result.Hostname = getString(cliConfig.Hostname, "CONTAINER_HOSTNAME", fileConfig.Hostname, "")
	result.LogDriver = getString(cliConfig.LogDriver, "LOG_DRIVER", fileConfig.LogDriver, "")

	// Boolean options
	result.Privileged = getBool(cliConfig.Privileged, "PRIVILEGED", fileConfig.Privileged)
	result.Init = getBool(cliConfig.Init, "INIT", fileConfig.Init)
	result.ReadOnly = getBool(cliConfig.ReadOnly, "READ_ONLY", fileConfig.ReadOnly)
	result.DryRun = getBool(cliConfig.DryRun, "DRY_RUN", fileConfig.DryRun)
	result.Verbose = getBool(cliConfig.Verbose, "VERBOSE", fileConfig.Verbose)
	result.Rollback = cliConfig.Rollback

	// Merge build args: file < env < CLI
	if result.BuildArgs == nil {
		result.BuildArgs = make(map[string]string)
	}
	if envBuildArgs := os.Getenv("DOCKER_BUILD_ARGS"); envBuildArgs != "" {
		for _, arg := range strings.Split(envBuildArgs, ",") {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				result.BuildArgs[parts[0]] = parts[1]
			}
		}
	}
	for _, arg := range buildArgs {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) == 2 {
			result.BuildArgs[parts[0]] = parts[1]
		}
	}

	// Merge labels: file < CLI
	if result.Labels == nil {
		result.Labels = make(map[string]string)
	}
	for _, label := range labelFlags {
		parts := strings.SplitN(label, "=", 2)
		if len(parts) == 2 {
			result.Labels[parts[0]] = parts[1]
		}
	}

	// Merge env: file < CLI
	if result.Env == nil {
		result.Env = make(map[string]string)
	}
	for _, env := range envFlags {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			result.Env[parts[0]] = parts[1]
		}
	}

	// Merge log opts: file < CLI
	if result.LogOpts == nil {
		result.LogOpts = make(map[string]string)
	}
	for _, opt := range logOptFlags {
		parts := strings.SplitN(opt, "=", 2)
		if len(parts) == 2 {
			result.LogOpts[parts[0]] = parts[1]
		}
	}

	// Merge slices: CLI > env > file
	if len(volumeFlags) > 0 {
		result.Volumes = []string(volumeFlags)
	} else if envVolumes := os.Getenv("DOCKER_VOLUMES"); envVolumes != "" {
		result.Volumes = strings.Split(envVolumes, ",")
	}

	if len(remoteCommandFlags) > 0 {
		result.RemoteCommands = []string(remoteCommandFlags)
	} else if envRemoteCommands := os.Getenv("REMOTE_COMMANDS"); envRemoteCommands != "" {
		result.RemoteCommands = strings.Split(envRemoteCommands, ",")
	}

	if len(extraHostFlags) > 0 {
		result.ExtraHosts = []string(extraHostFlags)
	}

	if len(capAddFlags) > 0 {
		result.CapAdd = []string(capAddFlags)
	}

	if len(capDropFlags) > 0 {
		result.CapDrop = []string(capDropFlags)
	}

	if len(tmpfsFlags) > 0 {
		result.Tmpfs = []string(tmpfsFlags)
	}

	return result
}

// This will be defined on build time
var version string

const helpText = `
Docker Deployment Tool

Usage:
  pipe [options]

Core Options:
  --config            Path to config file (default: pipe.yaml or pipe.yml)
  --host              Remote host to deploy to
  --user              SSH user for remote host
  --ssh-port          SSH port (default: 22)
  --ssh-key           Path to SSH key
  --dry-run           Preview deployment without executing

Image Options:
  --image             Docker image name (default: app)
  --dockerfile        Path to the Dockerfile (default: Dockerfile)
  --tag               Docker image tag (default: latest)
  --platform          Docker platform (default: linux/amd64)
  --build-arg         Build argument KEY=VALUE (can be repeated)

Container Options:
  --container-name    Name for the container (default: app)
  --container-port    Container port (default: 3000)
  --host-port         Host port (default: 3000)
  --env-file          Environment file path
  --env               Environment variable KEY=VALUE (can be repeated)
  --network           Docker network to connect to
  --volume            Volume mount host:container (can be repeated)
  --restart           Restart policy: no, always, on-failure, unless-stopped (default: unless-stopped)

Resource Limits:
  --cpus              Number of CPUs (e.g., '0.5' or '2')
  --memory            Memory limit (e.g., '512m' or '2g')

Health Check:
  --health-cmd        Health check command (e.g., 'curl -f http://localhost/')
  --health-interval   Time between checks (e.g., '30s')
  --health-timeout    Check timeout (e.g., '10s')
  --health-retries    Consecutive failures needed (e.g., 3)
  --health-start-period  Start period for container init (e.g., '5s')

Advanced Container Options:
  --entrypoint        Override container entrypoint
  --command           Override container command
  --container-user    User to run as (user or user:group)
  --workdir           Working directory inside container
  --hostname          Container hostname
  --add-host          Add host:ip mapping (can be repeated)
  --label             Container label KEY=VALUE (can be repeated)

Security Options:
  --privileged        Run container in privileged mode
  --read-only         Mount root filesystem as read-only
  --init              Run init inside container
  --cap-add           Add Linux capability (can be repeated)
  --cap-drop          Drop Linux capability (can be repeated)

Storage Options:
  --tmpfs             Mount tmpfs path or path:opts (can be repeated)

Logging Options:
  --log-driver        Logging driver (json-file, syslog, none, etc.)
  --log-opt           Log driver option KEY=VALUE (can be repeated)

Remote Execution:
  --remote-command    Command to run after deployment (can be repeated)

Other:
  --verbose, -v       Show detailed output
  --rollback          Rollback to the previous version
  --version           Show version information
  --help              Show this help message

Config File:
  Pipe automatically loads configuration from pipe.yaml or pipe.yml in the current
  directory. You can also specify a custom config file with --config.

  Priority order (highest to lowest):
    1. Command line flags
    2. Environment variables
    3. Config file
    4. Default values

  Example pipe.yaml:
    host: example.com
    user: deploy
    image: my-app
    tag: latest
    containerPort: "3000"
    hostPort: "3000"
    restartPolicy: unless-stopped
    
    healthCmd: "curl -f http://localhost:3000/health"
    healthInterval: "30s"
    healthTimeout: "10s"
    healthRetries: 3
    
    env:
      NODE_ENV: production
      LOG_LEVEL: info
    
    labels:
      app: my-app
      environment: production
    
    buildArgs:
      VERSION: "1.0.0"
      GIT_HASH: ${GIT_HASH}
    
    remoteCommands:
      - "docker system prune -f"

Environment Variables:
  HOST                        Remote host to deploy to
  HOST_USER                   SSH user for remote host
  HOST_PORT                   Host port
  HOST_PLATFORM               Docker platform
  SSH_PORT                    SSH port
  SSH_KEY_PATH                Path to SSH key
  DOCKERFILE                  Path to Dockerfile
  DOCKER_IMAGE_NAME           Docker image name
  DOCKER_IMAGE_TAG            Docker image tag
  DOCKER_CONTAINER_NAME       Name for the container
  DOCKER_CONTAINER_PORT       Container port
  DOCKER_CONTAINER_ENV_FILE   Environment file
  DOCKER_BUILD_ARGS           Build arguments (comma-separated KEY=VALUE)
  DOCKER_NETWORK              Docker network to connect to
  DOCKER_VOLUMES              Volume mounts (comma-separated)
  DOCKER_CPUS                 Number of CPUs
  DOCKER_MEMORY               Memory limit
  RESTART_POLICY              Restart policy
  HEALTH_CMD                  Health check command
  HEALTH_INTERVAL             Health check interval
  HEALTH_TIMEOUT              Health check timeout
  HEALTH_RETRIES              Health check retries
  HEALTH_START_PERIOD         Health check start period
  CONTAINER_USER              Container user
  CONTAINER_HOSTNAME          Container hostname
  WORKDIR                     Working directory
  LOG_DRIVER                  Log driver
  PRIVILEGED                  Privileged mode (true/false)
  READ_ONLY                   Read-only root fs (true/false)
  INIT                        Use init (true/false)
  REMOTE_COMMANDS             Remote commands (comma-separated)
  DRY_RUN                     Dry run mode (true/false)

Examples:
  # Deploy using config file
  pipe

  # Preview what would happen
  pipe --dry-run

  # Deploy with health check
  pipe --health-cmd "curl -f http://localhost:3000/health" --health-interval 30s

  # Deploy with environment variables
  pipe --env NODE_ENV=production --env LOG_LEVEL=debug

  # Deploy with labels
  pipe --label app=myapp --label version=1.0

  # Deploy with custom restart policy
  pipe --restart on-failure

  # Deploy with capabilities
  pipe --cap-add NET_ADMIN --cap-drop MKNOD

  # Deploy read-only container with tmpfs
  pipe --read-only --tmpfs /tmp --tmpfs /var/run

  # Rollback to previous version
  pipe --rollback
` 