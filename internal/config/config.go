package config

import (
	"errors"
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
	msgs := c.validateRequired()
	msgs = append(msgs, c.validateFormats()...)
	msgs = append(msgs, c.validatePaths()...)
	msgs = append(msgs, c.validateMaps()...)
	msgs = append(msgs, c.validateSlices()...)
	msgs = append(msgs, c.validateContainerOptions()...)
	msgs = append(msgs, c.validateRemoteCommands()...)

	if len(msgs) == 0 {
		return nil
	}

	errs := make([]error, len(msgs))
	for i, msg := range msgs {
		errs[i] = errors.New(msg)
	}
	return errors.Join(errs...)
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

	validateMap := func(m map[string]string, keyRegex *regexp.Regexp, name string) {
		for key, value := range m {
			if !keyRegex.MatchString(key) {
				errs = append(errs, fmt.Sprintf("%s key '%s' contains invalid characters", name, key))
			}
			if dangerousCharsRegex.MatchString(value) {
				errs = append(errs, fmt.Sprintf("%s value for '%s' contains dangerous shell characters", name, key))
			}
		}
	}

	validateMap(c.BuildArgs, buildArgKeyRegex, "build-arg")
	validateMap(c.Env, buildArgKeyRegex, "env")
	validateMap(c.Labels, labelKeyRegex, "label")
	validateMap(c.LogOpts, logOptKeyRegex, "log-opt")

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

	validateCaps := func(caps []string, name string) {
		for _, cap := range caps {
			if cap != "" && !capabilityRegex.MatchString(cap) {
				errs = append(errs, fmt.Sprintf("%s '%s' is not a valid Linux capability name", name, cap))
			}
		}
	}
	validateCaps(c.CapAdd, "cap-add")
	validateCaps(c.CapDrop, "cap-drop")

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

	checkDangerousChars := func(value, name string) {
		if value != "" && dangerousCharsRegex.MatchString(value) {
			errs = append(errs, name+" contains dangerous shell characters")
		}
	}

	checkDangerousChars(c.HealthCmd, "health-cmd")
	checkDangerousChars(c.Command, "command")
	checkDangerousChars(c.Entrypoint, "entrypoint")
	checkDangerousChars(c.ContainerUser, "container-user")
	checkDangerousChars(c.HealthInterval, "health-interval")
	checkDangerousChars(c.HealthTimeout, "health-timeout")
	checkDangerousChars(c.HealthStart, "health-start-period")

	if c.Workdir != "" {
		if !strings.HasPrefix(c.Workdir, "/") {
			errs = append(errs, "workdir must be an absolute path")
		}
		if strings.Contains(c.Workdir, "..") {
			errs = append(errs, "workdir contains path traversal sequence")
		}
		checkDangerousChars(c.Workdir, "workdir")
	}

	if c.Hostname != "" && !hostnameRegex.MatchString(c.Hostname) {
		errs = append(errs, "hostname contains invalid characters")
	}

	validRestartPolicies := map[string]bool{
		"no": true, "always": true, "on-failure": true, "unless-stopped": true,
	}
	if c.RestartPolicy != "" && !validRestartPolicies[c.RestartPolicy] {
		errs = append(errs, "restart policy must be one of: no, always, on-failure, unless-stopped")
	}

	if c.LogDriver != "" {
		validLogDrivers := map[string]bool{
			"json-file": true, "syslog": true, "journald": true, "gelf": true,
			"fluentd": true, "awslogs": true, "splunk": true, "gcplogs": true,
			"local": true, "none": true,
		}
		if !validLogDrivers[c.LogDriver] {
			errs = append(errs, fmt.Sprintf("log-driver '%s' is not a recognized Docker log driver", c.LogDriver))
		}
	}

	return errs
}
