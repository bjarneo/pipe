package cmd

import (
	"os"
	"reflect"
	"strings"

	"github.com/bjarneo/pipe/internal/config"
	"gopkg.in/yaml.v3"
)

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
	"LogFile":       "deploy.log",
}

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
	"LogFile":        "LOG_FILE",
}

func loadConfigFile(configPath string) config.Config {
	var cfg config.Config
	cfg.BuildArgs = make(map[string]string)
	cfg.Labels = make(map[string]string)
	cfg.Env = make(map[string]string)
	cfg.LogOpts = make(map[string]string)

	configPath = findConfigFile(configPath)
	if configPath == "" {
		return cfg
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return cfg
	}

	_ = yaml.Unmarshal(data, &cfg)
	expandEnvVars(&cfg)
	return cfg
}

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

func expandEnvVars(cfg *config.Config) {
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

func mergeConfigs(fileConfig, cliConfig config.Config) config.Config {
	result := fileConfig

	// Merge string fields
	rv := reflect.ValueOf(&result).Elem()
	cv := reflect.ValueOf(&cliConfig).Elem()
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

	// Merge int fields
	if cliConfig.HealthRetries != 0 {
		result.HealthRetries = cliConfig.HealthRetries
	}

	// Merge bool fields
	result.Privileged = mergeBool(cliConfig.Privileged, "PRIVILEGED", fileConfig.Privileged)
	result.Init = mergeBool(cliConfig.Init, "INIT", fileConfig.Init)
	result.ReadOnly = mergeBool(cliConfig.ReadOnly, "READ_ONLY", fileConfig.ReadOnly)
	result.DryRun = mergeBool(cliConfig.DryRun, "DRY_RUN", fileConfig.DryRun)
	result.Verbose = mergeBool(cliConfig.Verbose, "VERBOSE", fileConfig.Verbose)
	result.JSONOutput = mergeBool(cliConfig.JSONOutput, "JSON_OUTPUT", fileConfig.JSONOutput)

	// Merge maps from file config
	if result.BuildArgs == nil {
		result.BuildArgs = make(map[string]string)
	}
	if result.Labels == nil {
		result.Labels = make(map[string]string)
	}
	if result.Env == nil {
		result.Env = make(map[string]string)
	}
	if result.LogOpts == nil {
		result.LogOpts = make(map[string]string)
	}

	// Build args from env
	if envBuildArgs := os.Getenv("DOCKER_BUILD_ARGS"); envBuildArgs != "" {
		for _, item := range strings.Split(envBuildArgs, ",") {
			parts := strings.SplitN(item, "=", 2)
			if len(parts) == 2 {
				result.BuildArgs[parts[0]] = parts[1]
			}
		}
	}

	// Volumes from env
	if len(result.Volumes) == 0 {
		if envVolumes := os.Getenv("DOCKER_VOLUMES"); envVolumes != "" {
			result.Volumes = strings.Split(envVolumes, ",")
		}
	}

	// Remote commands from env
	if len(result.RemoteCommands) == 0 {
		if envRemoteCommands := os.Getenv("REMOTE_COMMANDS"); envRemoteCommands != "" {
			result.RemoteCommands = strings.Split(envRemoteCommands, ",")
		}
	}

	return result
}

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
