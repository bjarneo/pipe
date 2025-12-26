package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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
	DryRun     bool `json:"dryRun" yaml:"dryRun"`
	Verbose    bool `json:"verbose" yaml:"verbose"`
	JSONOutput bool `json:"jsonOutput" yaml:"jsonOutput"`
	ShowStats  bool `json:"showStats" yaml:"showStats"`
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

// flagSet holds all CLI flags for parsing
type flagSet struct {
	buildArgs          arrayFlags
	volumeFlags        arrayFlags
	remoteCommandFlags arrayFlags
	envFlags           arrayFlags
	labelFlags         arrayFlags
	extraHostFlags     arrayFlags
	capAddFlags        arrayFlags
	capDropFlags       arrayFlags
	tmpfsFlags         arrayFlags
	logOptFlags        arrayFlags
	configFile         string
	showHelp           bool
	showVersion        bool
}

// Load loads configuration from command line flags and environment variables
func Load() Config {
	var config Config
	var flags flagSet

	initMaps(&config)
	defineFlags(&config, &flags)
	flag.Parse()

	handleHelpAndVersion(flags)

	fileConfig := loadConfigFile(flags.configFile)
	config = mergeConfig(fileConfig, config, flags)
	config.SSHKey = expandHomePath(config.SSHKey)

	return config
}

// initMaps initializes all map fields to prevent nil map assignments
func initMaps(cfg *Config) {
	cfg.BuildArgs = make(map[string]string)
	cfg.Labels = make(map[string]string)
	cfg.Env = make(map[string]string)
	cfg.LogOpts = make(map[string]string)
}

// defineFlags sets up all command line flags
func defineFlags(cfg *Config, flags *flagSet) {
	// Config file
	flag.StringVar(&flags.configFile, "config", "", "Path to config file (default: pipe.yaml or pipe.yml)")

	// Core options
	flag.StringVar(&cfg.Host, "host", "", "Remote host to deploy to")
	flag.StringVar(&cfg.User, "user", "", "SSH user for remote host")
	flag.StringVar(&cfg.SSHPort, "ssh-port", "", "SSH port (default: 22)")
	flag.StringVar(&cfg.SSHKey, "ssh-key", "", "Path to SSH key")

	// Image options
	flag.StringVar(&cfg.Image, "image", "", "Docker image name")
	flag.StringVar(&cfg.Dockerfile, "dockerfile", "", "Path to the Dockerfile")
	flag.StringVar(&cfg.Tag, "tag", "", "Docker image tag")
	flag.StringVar(&cfg.Platform, "platform", "", "Docker platform")
	flag.Var(&flags.buildArgs, "build-arg", "Build argument in KEY=VALUE format (can be specified multiple times)")

	// Container options
	flag.StringVar(&cfg.ContainerName, "container-name", "", "Name for the container")
	flag.StringVar(&cfg.ContainerPort, "container-port", "", "Container port")
	flag.StringVar(&cfg.HostPort, "host-port", "", "Host port")
	flag.StringVar(&cfg.EnvFile, "env-file", "", "Environment file")
	flag.Var(&flags.volumeFlags, "volume", "Volume mount in format 'host:container' (can be specified multiple times)")
	flag.Var(&flags.remoteCommandFlags, "remote-command", "Remote command to execute after deployment (can be specified multiple times)")
	flag.StringVar(&cfg.Network, "network", "", "Docker network to connect to")

	// Resource limits
	flag.StringVar(&cfg.CPUs, "cpus", "", "Number of CPUs (e.g., '0.5' or '2')")
	flag.StringVar(&cfg.Memory, "memory", "", "Memory limit (e.g., '512m' or '2g')")

	// Health check flags
	flag.StringVar(&cfg.HealthCmd, "health-cmd", "", "Health check command")
	flag.StringVar(&cfg.HealthInterval, "health-interval", "", "Health check interval (e.g., '30s')")
	flag.StringVar(&cfg.HealthTimeout, "health-timeout", "", "Health check timeout (e.g., '10s')")
	flag.IntVar(&cfg.HealthRetries, "health-retries", 0, "Health check retries")
	flag.StringVar(&cfg.HealthStart, "health-start-period", "", "Health check start period (e.g., '5s')")

	// Container runtime flags
	flag.StringVar(&cfg.RestartPolicy, "restart", "", "Restart policy (no, always, on-failure, unless-stopped)")
	flag.Var(&flags.labelFlags, "label", "Container label in KEY=VALUE format (can be specified multiple times)")
	flag.Var(&flags.envFlags, "env", "Environment variable in KEY=VALUE format (can be specified multiple times)")
	flag.StringVar(&cfg.Entrypoint, "entrypoint", "", "Override container entrypoint")
	flag.StringVar(&cfg.Command, "command", "", "Override container command")
	flag.StringVar(&cfg.ContainerUser, "container-user", "", "User to run container as (user or user:group)")
	flag.StringVar(&cfg.Workdir, "workdir", "", "Working directory inside container")
	flag.StringVar(&cfg.Hostname, "hostname", "", "Container hostname")
	flag.Var(&flags.extraHostFlags, "add-host", "Add host-to-IP mapping (host:ip)")

	// Security flags
	flag.BoolVar(&cfg.Privileged, "privileged", false, "Run container in privileged mode")
	flag.BoolVar(&cfg.Init, "init", false, "Run init inside container")
	flag.BoolVar(&cfg.ReadOnly, "read-only", false, "Mount root filesystem as read-only")
	flag.Var(&flags.capAddFlags, "cap-add", "Add Linux capability")
	flag.Var(&flags.capDropFlags, "cap-drop", "Drop Linux capability")

	// Storage flags
	flag.Var(&flags.tmpfsFlags, "tmpfs", "Mount tmpfs (path or path:opts)")

	// Logging flags
	flag.StringVar(&cfg.LogDriver, "log-driver", "", "Logging driver (e.g., json-file, syslog, none)")
	flag.Var(&flags.logOptFlags, "log-opt", "Log driver options in KEY=VALUE format")

	// Execution flags
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "Preview deployment without executing")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "Show detailed output")
	flag.BoolVar(&cfg.Verbose, "v", false, "Show detailed output (shorthand)")
	flag.BoolVar(&cfg.JSONOutput, "json", false, "Output deployment stats as JSON")
	flag.BoolVar(&cfg.ShowStats, "stats", false, "Show container stats from remote host")
	flag.BoolVar(&flags.showHelp, "help", false, "Show help message")
	flag.BoolVar(&cfg.Rollback, "rollback", false, "Rollback to previous version")
	flag.BoolVar(&flags.showVersion, "version", false, "Show version information")

	flag.Usage = func() { fmt.Print(helpText) }
}

// handleHelpAndVersion exits if help or version flags are set
func handleHelpAndVersion(flags flagSet) {
	if flags.showHelp {
		flag.Usage()
		os.Exit(0)
	}
	if flags.showVersion {
		fmt.Printf("pipe version %s\n", version)
		os.Exit(0)
	}
}

// expandHomePath expands ~ to home directory in paths
func expandHomePath(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to expand ~ in path: %v\n", err)
		return path
	}
	return filepath.Join(home, path[2:])
}

// =============================================================================
// Validation
// =============================================================================

// Validation patterns - prevent shell injection with strict regexes
var (
	hostnameRegex       = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-\.]*[a-zA-Z0-9])?$`)
	ipRegex             = regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
	usernameRegex       = regexp.MustCompile(`^[a-z_][a-z0-9_-]*$`)
	imageNameRegex      = regexp.MustCompile(`^[a-z0-9][a-z0-9._/-]*$`)
	tagRegex            = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)
	containerNameRegex  = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)
	networkNameRegex    = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)
	buildArgKeyRegex    = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	labelKeyRegex       = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)
	logOptKeyRegex      = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)
	capabilityRegex     = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
	cpuRegex            = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?$`)
	memoryRegex         = regexp.MustCompile(`^[0-9]+[bkmgBKMG]?$`)
	dangerousCharsRegex = regexp.MustCompile(`[;&|$` + "`" + `\\\n\r"'<>(){}]`)
)

// Port range constants
const (
	minPort = 1
	maxPort = 65535
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

// Validate validates the configuration and returns an error if any field is invalid.
// This is critical for preventing command injection attacks.
func (c *Config) Validate() error {
	var errs []string

	errs = append(errs, c.validateRequired()...)
	errs = append(errs, c.validateFormats()...)
	errs = append(errs, c.validatePaths()...)
	errs = append(errs, c.validateMaps()...)
	errs = append(errs, c.validateSlices()...)
	errs = append(errs, c.validateContainerOptions()...)
	errs = append(errs, c.validateRemoteCommands()...)

	if len(errs) > 0 {
		return fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// validateRequired checks required fields are present and valid
func (c *Config) validateRequired() []string {
	var errs []string

	if c.Host == "" {
		errs = append(errs, "host is required")
	} else if !hostnameRegex.MatchString(c.Host) && !ipRegex.MatchString(c.Host) {
		errs = append(errs, "host must be a valid hostname or IP address")
	}

	if c.User == "" {
		errs = append(errs, "user is required")
	} else if !usernameRegex.MatchString(c.User) {
		errs = append(errs, "user must be a valid Unix username (lowercase, starts with letter/underscore)")
	}

	return errs
}

// validateFormats checks format constraints on optional fields
func (c *Config) validateFormats() []string {
	var errs []string

	if c.Image != "" && !imageNameRegex.MatchString(c.Image) {
		errs = append(errs, "image name contains invalid characters")
	}
	if c.Tag != "" && !tagRegex.MatchString(c.Tag) {
		errs = append(errs, "tag contains invalid characters")
	}
	if c.ContainerName != "" && !containerNameRegex.MatchString(c.ContainerName) {
		errs = append(errs, "container-name contains invalid characters")
	}
	if c.Platform != "" && !validPlatforms[c.Platform] {
		errs = append(errs, "platform must be one of: linux/amd64, linux/arm64, linux/arm/v7, linux/arm/v6, linux/386")
	}
	if c.Network != "" && !networkNameRegex.MatchString(c.Network) {
		errs = append(errs, "network name contains invalid characters")
	}
	if c.CPUs != "" && !cpuRegex.MatchString(c.CPUs) {
		errs = append(errs, "cpus must be a valid number (e.g., '0.5' or '2')")
	}
	if c.Memory != "" && !memoryRegex.MatchString(c.Memory) {
		errs = append(errs, "memory must be a valid format (e.g., '512m' or '2g')")
	}

	errs = append(errs, validatePort(c.ContainerPort, "container-port")...)
	errs = append(errs, validatePort(c.HostPort, "host-port")...)

	return errs
}

// validatePort validates a port number string
func validatePort(port, name string) []string {
	if port == "" {
		return nil
	}
	if p, err := strconv.Atoi(port); err != nil || p < minPort || p > maxPort {
		return []string{fmt.Sprintf("%s must be a valid port number (%d-%d)", name, minPort, maxPort)}
	}
	return nil
}

// validatePaths checks file paths for security issues
func (c *Config) validatePaths() []string {
	var errs []string

	if c.EnvFile != "" {
		if strings.Contains(c.EnvFile, "..") {
			errs = append(errs, "env-file path contains path traversal sequence")
		}
		if filepath.IsAbs(c.EnvFile) {
			errs = append(errs, "env-file must be a relative path")
		}
		if dangerousCharsRegex.MatchString(c.EnvFile) {
			errs = append(errs, "env-file path contains dangerous shell characters")
		}
	}

	if c.Dockerfile != "" {
		if strings.Contains(c.Dockerfile, "..") {
			errs = append(errs, "dockerfile path contains path traversal sequence")
		}
		if dangerousCharsRegex.MatchString(c.Dockerfile) {
			errs = append(errs, "dockerfile path contains dangerous shell characters")
		}
	}

	if c.SSHKey != "" && dangerousCharsRegex.MatchString(c.SSHKey) {
		errs = append(errs, "ssh-key path contains dangerous shell characters")
	}

	return errs
}

// validateMaps checks map fields for shell injection
func (c *Config) validateMaps() []string {
	var errs []string

	// Validate BuildArgs
	for key, value := range c.BuildArgs {
		if !buildArgKeyRegex.MatchString(key) {
			errs = append(errs, fmt.Sprintf("build-arg key '%s' contains invalid characters", key))
		}
		if dangerousCharsRegex.MatchString(value) {
			errs = append(errs, fmt.Sprintf("build-arg value for '%s' contains dangerous shell characters", key))
		}
	}

	// Validate Env
	for key, value := range c.Env {
		if !buildArgKeyRegex.MatchString(key) {
			errs = append(errs, fmt.Sprintf("env key '%s' contains invalid characters", key))
		}
		if dangerousCharsRegex.MatchString(value) {
			errs = append(errs, fmt.Sprintf("env value for '%s' contains dangerous shell characters", key))
		}
	}

	// Validate Labels
	for key, value := range c.Labels {
		if !labelKeyRegex.MatchString(key) {
			errs = append(errs, fmt.Sprintf("label key '%s' contains invalid characters", key))
		}
		if dangerousCharsRegex.MatchString(value) {
			errs = append(errs, fmt.Sprintf("label value for '%s' contains dangerous shell characters", key))
		}
	}

	// Validate LogOpts
	for key, value := range c.LogOpts {
		if !logOptKeyRegex.MatchString(key) {
			errs = append(errs, fmt.Sprintf("log-opt key '%s' contains invalid characters", key))
		}
		if dangerousCharsRegex.MatchString(value) {
			errs = append(errs, fmt.Sprintf("log-opt value for '%s' contains dangerous shell characters", key))
		}
	}

	return errs
}

// validateSlices checks slice fields for security issues
func (c *Config) validateSlices() []string {
	var errs []string

	// Validate Volumes
	for _, vol := range c.Volumes {
		if vol == "" {
			continue
		}
		if !strings.Contains(vol, ":") {
			errs = append(errs, fmt.Sprintf("volume '%s' must be in format 'host:container'", vol))
		}
		if dangerousCharsRegex.MatchString(vol) {
			errs = append(errs, fmt.Sprintf("volume '%s' contains dangerous shell characters", vol))
		}
		if strings.Contains(vol, "..") {
			errs = append(errs, fmt.Sprintf("volume '%s' contains path traversal sequence", vol))
		}
	}

	// Validate ExtraHosts (format: hostname:ip)
	for _, host := range c.ExtraHosts {
		if host == "" {
			continue
		}
		if !strings.Contains(host, ":") {
			errs = append(errs, fmt.Sprintf("extra-host '%s' must be in format 'hostname:ip'", host))
		} else {
			parts := strings.SplitN(host, ":", 2)
			if !hostnameRegex.MatchString(parts[0]) {
				errs = append(errs, fmt.Sprintf("extra-host '%s' has invalid hostname", host))
			}
			if !ipRegex.MatchString(parts[1]) {
				errs = append(errs, fmt.Sprintf("extra-host '%s' has invalid IP address", host))
			}
		}
	}

	// Validate CapAdd
	for _, cap := range c.CapAdd {
		if cap == "" {
			continue
		}
		if !capabilityRegex.MatchString(cap) {
			errs = append(errs, fmt.Sprintf("cap-add '%s' is not a valid Linux capability name", cap))
		}
	}

	// Validate CapDrop
	for _, cap := range c.CapDrop {
		if cap == "" {
			continue
		}
		if !capabilityRegex.MatchString(cap) {
			errs = append(errs, fmt.Sprintf("cap-drop '%s' is not a valid Linux capability name", cap))
		}
	}

	// Validate Tmpfs
	for _, tmpfs := range c.Tmpfs {
		if tmpfs == "" {
			continue
		}
		// Tmpfs can be just a path or path:options
		path := tmpfs
		if strings.Contains(tmpfs, ":") {
			path = strings.SplitN(tmpfs, ":", 2)[0]
		}
		if !strings.HasPrefix(path, "/") {
			errs = append(errs, fmt.Sprintf("tmpfs '%s' must be an absolute path", tmpfs))
		}
		if strings.Contains(path, "..") {
			errs = append(errs, fmt.Sprintf("tmpfs '%s' contains path traversal sequence", tmpfs))
		}
		if dangerousCharsRegex.MatchString(tmpfs) {
			errs = append(errs, fmt.Sprintf("tmpfs '%s' contains dangerous shell characters", tmpfs))
		}
	}

	return errs
}

// validateRemoteCommands checks remote commands for dangerous patterns
func (c *Config) validateRemoteCommands() []string {
	var errs []string
	dangerousPatterns := []string{"rm -rf /", "mkfs", "dd if=", "> /dev/"}

	for i, cmd := range c.RemoteCommands {
		if cmd == "" {
			continue
		}
		for _, pattern := range dangerousPatterns {
			if strings.Contains(cmd, pattern) {
				errs = append(errs, fmt.Sprintf("remote-command[%d] contains dangerous pattern '%s'", i, pattern))
			}
		}
	}

	return errs
}

// validateContainerOptions checks container runtime options for security issues
func (c *Config) validateContainerOptions() []string {
	var errs []string

	// Validate HealthCmd - executed in shell, check for dangerous characters
	if c.HealthCmd != "" {
		if dangerousCharsRegex.MatchString(c.HealthCmd) {
			errs = append(errs, "health-cmd contains dangerous shell characters")
		}
	}

	// Validate Command - passed to container, check for dangerous characters
	if c.Command != "" {
		if dangerousCharsRegex.MatchString(c.Command) {
			errs = append(errs, "command contains dangerous shell characters")
		}
	}

	// Validate Entrypoint - passed to container, check for dangerous characters
	if c.Entrypoint != "" {
		if dangerousCharsRegex.MatchString(c.Entrypoint) {
			errs = append(errs, "entrypoint contains dangerous shell characters")
		}
	}

	// Validate Workdir - must be absolute path
	if c.Workdir != "" {
		if !strings.HasPrefix(c.Workdir, "/") {
			errs = append(errs, "workdir must be an absolute path")
		}
		if strings.Contains(c.Workdir, "..") {
			errs = append(errs, "workdir contains path traversal sequence")
		}
		if dangerousCharsRegex.MatchString(c.Workdir) {
			errs = append(errs, "workdir contains dangerous shell characters")
		}
	}

	// Validate Hostname
	if c.Hostname != "" {
		if !hostnameRegex.MatchString(c.Hostname) {
			errs = append(errs, "hostname contains invalid characters")
		}
	}

	// Validate ContainerUser (format: user or user:group)
	if c.ContainerUser != "" {
		if dangerousCharsRegex.MatchString(c.ContainerUser) {
			errs = append(errs, "container-user contains dangerous shell characters")
		}
	}

	// Validate RestartPolicy
	validRestartPolicies := map[string]bool{
		"no":             true,
		"always":         true,
		"on-failure":     true,
		"unless-stopped": true,
	}
	if c.RestartPolicy != "" && !validRestartPolicies[c.RestartPolicy] {
		errs = append(errs, "restart policy must be one of: no, always, on-failure, unless-stopped")
	}

	// Validate LogDriver
	if c.LogDriver != "" {
		validLogDrivers := map[string]bool{
			"json-file": true,
			"syslog":    true,
			"journald":  true,
			"gelf":      true,
			"fluentd":   true,
			"awslogs":   true,
			"splunk":    true,
			"gcplogs":   true,
			"local":     true,
			"none":      true,
		}
		if !validLogDrivers[c.LogDriver] {
			errs = append(errs, fmt.Sprintf("log-driver '%s' is not a recognized Docker log driver", c.LogDriver))
		}
	}

	// Validate health check timing formats
	if c.HealthInterval != "" {
		if dangerousCharsRegex.MatchString(c.HealthInterval) {
			errs = append(errs, "health-interval contains dangerous shell characters")
		}
	}
	if c.HealthTimeout != "" {
		if dangerousCharsRegex.MatchString(c.HealthTimeout) {
			errs = append(errs, "health-timeout contains dangerous shell characters")
		}
	}
	if c.HealthStart != "" {
		if dangerousCharsRegex.MatchString(c.HealthStart) {
			errs = append(errs, "health-start-period contains dangerous shell characters")
		}
	}

	return errs
}

// =============================================================================
// Configuration Loading
// =============================================================================

// loadConfigFile loads configuration from a YAML file
func loadConfigFile(configPath string) Config {
	var config Config
	initMaps(&config)

	configPath = findConfigFile(configPath)
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

	expandEnvVars(&config)
	return config
}

// findConfigFile returns the config file path, checking defaults if needed
func findConfigFile(path string) string {
	if path != "" {
		return path
	}
	for _, name := range []string{"pipe.yaml", "pipe.yml"} {
		if _, err := os.Stat(name); err == nil {
			return name
		}
	}
	return ""
}

// expandEnvVars expands environment variables in all string fields using reflection
func expandEnvVars(cfg *Config) {
	expand := func(s string) string {
		return os.Expand(s, func(key string) string {
			if val, exists := os.LookupEnv(key); exists {
				return val
			}
			return "${" + key + "}"
		})
	}

	v := reflect.ValueOf(cfg).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.CanSet() {
			continue
		}

		switch field.Kind() {
		case reflect.String:
			field.SetString(expand(field.String()))

		case reflect.Slice:
			if field.Type().Elem().Kind() == reflect.String {
				for j := 0; j < field.Len(); j++ {
					field.Index(j).SetString(expand(field.Index(j).String()))
				}
			}

		case reflect.Map:
			if t.Field(i).Type.Key().Kind() == reflect.String &&
				t.Field(i).Type.Elem().Kind() == reflect.String {
				for _, key := range field.MapKeys() {
					val := field.MapIndex(key).String()
					field.SetMapIndex(key, reflect.ValueOf(expand(val)))
				}
			}
		}
	}
}

// =============================================================================
// Configuration Merging
// =============================================================================

// envMapping defines environment variable to config field mappings
var envMapping = map[string]string{
	"Host":           "HOST",
	"User":           "HOST_USER",
	"SSHPort":        "SSH_PORT",
	"Image":          "DOCKER_IMAGE_NAME",
	"Dockerfile":     "DOCKERFILE",
	"Tag":            "DOCKER_IMAGE_TAG",
	"Platform":       "HOST_PLATFORM",
	"SSHKey":         "SSH_KEY_PATH",
	"ContainerName":  "DOCKER_CONTAINER_NAME",
	"ContainerPort":  "DOCKER_CONTAINER_PORT",
	"HostPort":       "HOST_PORT",
	"EnvFile":        "DOCKER_CONTAINER_ENV_FILE",
	"Network":        "DOCKER_NETWORK",
	"CPUs":           "DOCKER_CPUS",
	"Memory":         "DOCKER_MEMORY",
	"HealthCmd":      "HEALTH_CMD",
	"HealthInterval": "HEALTH_INTERVAL",
	"HealthTimeout":  "HEALTH_TIMEOUT",
	"HealthStart":    "HEALTH_START_PERIOD",
	"RestartPolicy":  "RESTART_POLICY",
	"ContainerUser":  "CONTAINER_USER",
	"Workdir":        "WORKDIR",
	"Hostname":       "CONTAINER_HOSTNAME",
	"LogDriver":      "LOG_DRIVER",
}

// defaultValues defines default values for config fields
var defaultValues = map[string]string{
	"SSHPort":       "22",
	"Image":         "app",
	"Dockerfile":    "Dockerfile",
	"Tag":           "latest",
	"Platform":      "linux/amd64",
	"ContainerName": "app",
	"ContainerPort": "3000",
	"HostPort":      "3000",
	"RestartPolicy": "unless-stopped",
}

// mergeConfig merges configuration from file, environment variables, and CLI flags
// Priority: CLI flags > env vars > config file > defaults
func mergeConfig(fileConfig, cliConfig Config, flags flagSet) Config {
	result := fileConfig

	// Merge string fields using reflection
	mergeStringFields(&result, &cliConfig)

	// Merge int fields
	result.HealthRetries = mergeInt(cliConfig.HealthRetries, "HEALTH_RETRIES", fileConfig.HealthRetries, 0)

	// Merge bool fields
	result.Privileged = mergeBool(cliConfig.Privileged, "PRIVILEGED", fileConfig.Privileged)
	result.Init = mergeBool(cliConfig.Init, "INIT", fileConfig.Init)
	result.ReadOnly = mergeBool(cliConfig.ReadOnly, "READ_ONLY", fileConfig.ReadOnly)
	result.DryRun = mergeBool(cliConfig.DryRun, "DRY_RUN", fileConfig.DryRun)
	result.Verbose = mergeBool(cliConfig.Verbose, "VERBOSE", fileConfig.Verbose)
	result.JSONOutput = mergeBool(cliConfig.JSONOutput, "JSON_OUTPUT", fileConfig.JSONOutput)
	result.Rollback = cliConfig.Rollback
	result.ShowStats = cliConfig.ShowStats

	// Merge maps
	mergeMaps(&result, flags)

	// Merge slices
	mergeSlices(&result, flags)

	return result
}

// mergeStringFields merges all string fields based on priority
func mergeStringFields(result, cliConfig *Config) {
	rv := reflect.ValueOf(result).Elem()
	cv := reflect.ValueOf(cliConfig).Elem()
	t := rv.Type()

	for i := 0; i < rv.NumField(); i++ {
		field := t.Field(i)
		if field.Type.Kind() != reflect.String {
			continue
		}

		fieldName := field.Name
		resultField := rv.Field(i)
		cliField := cv.Field(i)

		cliVal := cliField.String()
		fileVal := resultField.String()
		envKey := envMapping[fieldName]
		defaultVal := defaultValues[fieldName]

		resultField.SetString(mergeString(cliVal, envKey, fileVal, defaultVal))
	}
}

// mergeString returns the value with highest priority
func mergeString(cliVal, envKey, fileVal, defaultVal string) string {
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

// mergeInt returns the value with highest priority
func mergeInt(cliVal int, envKey string, fileVal, defaultVal int) int {
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

// mergeBool returns the value with highest priority
func mergeBool(cliVal bool, envKey string, fileVal bool) bool {
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

// mergeMaps merges map fields from various sources
func mergeMaps(result *Config, flags flagSet) {
	ensureMap(&result.BuildArgs)
	ensureMap(&result.Labels)
	ensureMap(&result.Env)
	ensureMap(&result.LogOpts)

	// Build args from env
	if envBuildArgs := os.Getenv("DOCKER_BUILD_ARGS"); envBuildArgs != "" {
		parseKeyValues(envBuildArgs, ",", result.BuildArgs)
	}

	// CLI flags override
	parseArrayFlags(flags.buildArgs, result.BuildArgs)
	parseArrayFlags(flags.labelFlags, result.Labels)
	parseArrayFlags(flags.envFlags, result.Env)
	parseArrayFlags(flags.logOptFlags, result.LogOpts)
}

// mergeSlices merges slice fields from CLI flags and env
func mergeSlices(result *Config, flags flagSet) {
	if len(flags.volumeFlags) > 0 {
		result.Volumes = []string(flags.volumeFlags)
	} else if envVolumes := os.Getenv("DOCKER_VOLUMES"); envVolumes != "" {
		result.Volumes = strings.Split(envVolumes, ",")
	}

	if len(flags.remoteCommandFlags) > 0 {
		result.RemoteCommands = []string(flags.remoteCommandFlags)
	} else if envRemoteCommands := os.Getenv("REMOTE_COMMANDS"); envRemoteCommands != "" {
		result.RemoteCommands = strings.Split(envRemoteCommands, ",")
	}

	if len(flags.extraHostFlags) > 0 {
		result.ExtraHosts = []string(flags.extraHostFlags)
	}
	if len(flags.capAddFlags) > 0 {
		result.CapAdd = []string(flags.capAddFlags)
	}
	if len(flags.capDropFlags) > 0 {
		result.CapDrop = []string(flags.capDropFlags)
	}
	if len(flags.tmpfsFlags) > 0 {
		result.Tmpfs = []string(flags.tmpfsFlags)
	}
}

// ensureMap ensures a map is initialized
func ensureMap(m *map[string]string) {
	if *m == nil {
		*m = make(map[string]string)
	}
}

// parseKeyValues parses "key=value,key=value" into a map
func parseKeyValues(input, sep string, target map[string]string) {
	for _, item := range strings.Split(input, sep) {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) == 2 {
			target[parts[0]] = parts[1]
		}
	}
}

// parseArrayFlags parses array flags into a map
func parseArrayFlags(flags arrayFlags, target map[string]string) {
	for _, item := range flags {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) == 2 {
			target[parts[0]] = parts[1]
		}
	}
}

// =============================================================================
// Version and Help
// =============================================================================

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
  --json              Output stats as JSON (for piping to other tools)
  --stats             Show container stats from remote host (CPU, memory, network, etc.)
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
  JSON_OUTPUT                 Output JSON stats (true/false)

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

  # Show container stats from remote host
  pipe --stats

  # Output container stats as JSON
  pipe --stats --json | jq '.resources'

  # Output deployment stats as JSON (for piping to other tools)
  pipe --json | jq '.duration'
`
