package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
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
	DryRun     bool   `json:"dryRun" yaml:"dryRun"`
	Verbose    bool   `json:"verbose" yaml:"verbose"`
	JSONOutput bool   `json:"jsonOutput" yaml:"jsonOutput"`
	ShowStats  bool   `json:"showStats" yaml:"showStats"`
	LogFile    string `json:"logFile" yaml:"logFile"`
}

// Validation patterns
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

const (
	minPort = 1
	maxPort = 65535
)

var validPlatforms = map[string]bool{
	"linux/amd64":   true,
	"linux/arm64":   true,
	"linux/arm/v7":  true,
	"linux/arm/v6":  true,
	"linux/386":     true,
	"linux/ppc64le": true,
	"linux/s390x":   true,
}

// Validate validates the configuration
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

func validatePort(port, name string) []string {
	if port == "" {
		return nil
	}
	if p, err := strconv.Atoi(port); err != nil || p < minPort || p > maxPort {
		return []string{fmt.Sprintf("%s must be a valid port number (%d-%d)", name, minPort, maxPort)}
	}
	return nil
}

func (c *Config) validatePaths() []string {
	var errs []string

	if c.EnvFile != "" {
		if strings.Contains(c.EnvFile, "..") {
			errs = append(errs, "env-file path contains path traversal sequence")
		}
		if strings.HasPrefix(c.EnvFile, "/") {
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

func (c *Config) validateMaps() []string {
	var errs []string

	for key, value := range c.BuildArgs {
		if !buildArgKeyRegex.MatchString(key) {
			errs = append(errs, fmt.Sprintf("build-arg key '%s' contains invalid characters", key))
		}
		if dangerousCharsRegex.MatchString(value) {
			errs = append(errs, fmt.Sprintf("build-arg value for '%s' contains dangerous shell characters", key))
		}
	}

	for key, value := range c.Env {
		if !buildArgKeyRegex.MatchString(key) {
			errs = append(errs, fmt.Sprintf("env key '%s' contains invalid characters", key))
		}
		if dangerousCharsRegex.MatchString(value) {
			errs = append(errs, fmt.Sprintf("env value for '%s' contains dangerous shell characters", key))
		}
	}

	for key, value := range c.Labels {
		if !labelKeyRegex.MatchString(key) {
			errs = append(errs, fmt.Sprintf("label key '%s' contains invalid characters", key))
		}
		if dangerousCharsRegex.MatchString(value) {
			errs = append(errs, fmt.Sprintf("label value for '%s' contains dangerous shell characters", key))
		}
	}

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

func (c *Config) validateSlices() []string {
	var errs []string

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

	for _, cap := range c.CapAdd {
		if cap == "" {
			continue
		}
		if !capabilityRegex.MatchString(cap) {
			errs = append(errs, fmt.Sprintf("cap-add '%s' is not a valid Linux capability name", cap))
		}
	}

	for _, cap := range c.CapDrop {
		if cap == "" {
			continue
		}
		if !capabilityRegex.MatchString(cap) {
			errs = append(errs, fmt.Sprintf("cap-drop '%s' is not a valid Linux capability name", cap))
		}
	}

	for _, tmpfs := range c.Tmpfs {
		if tmpfs == "" {
			continue
		}
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

func (c *Config) validateContainerOptions() []string {
	var errs []string

	if c.HealthCmd != "" {
		if dangerousCharsRegex.MatchString(c.HealthCmd) {
			errs = append(errs, "health-cmd contains dangerous shell characters")
		}
	}

	if c.Command != "" {
		if dangerousCharsRegex.MatchString(c.Command) {
			errs = append(errs, "command contains dangerous shell characters")
		}
	}

	if c.Entrypoint != "" {
		if dangerousCharsRegex.MatchString(c.Entrypoint) {
			errs = append(errs, "entrypoint contains dangerous shell characters")
		}
	}

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

	if c.Hostname != "" {
		if !hostnameRegex.MatchString(c.Hostname) {
			errs = append(errs, "hostname contains invalid characters")
		}
	}

	if c.ContainerUser != "" {
		if dangerousCharsRegex.MatchString(c.ContainerUser) {
			errs = append(errs, "container-user contains dangerous shell characters")
		}
	}

	validRestartPolicies := map[string]bool{
		"no":             true,
		"always":         true,
		"on-failure":     true,
		"unless-stopped": true,
	}
	if c.RestartPolicy != "" && !validRestartPolicies[c.RestartPolicy] {
		errs = append(errs, "restart policy must be one of: no, always, on-failure, unless-stopped")
	}

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
