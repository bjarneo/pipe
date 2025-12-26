package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bjarneo/pipe/internal/config"
	"github.com/bjarneo/pipe/internal/container"
	"github.com/bjarneo/pipe/internal/deploy"
	"github.com/bjarneo/pipe/internal/logger"
	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	cfg       config.Config
	cfgFile   string
	sliceOpts sliceFlags
)

type sliceFlags struct {
	buildArgs      []string
	volumes        []string
	remoteCommands []string
	envVars        []string
	labels         []string
	extraHosts     []string
	capAdd         []string
	capDrop        []string
	tmpfs          []string
	logOpts        []string
}

var rootCmd = &cobra.Command{
	Use:   "pipe",
	Short: "Docker deployment CLI tool",
	Long: `Pipe is a Docker deployment CLI tool that transfers Docker images 
to remote hosts via SSH without requiring a registry. 
It uses delta transfers to only send changed layers.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfig()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDeploy()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Config file
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: pipe.yaml)")

	// Core options
	rootCmd.PersistentFlags().StringVar(&cfg.Host, "host", "", "remote host to deploy to")
	rootCmd.PersistentFlags().StringVar(&cfg.User, "user", "", "SSH user for remote host")
	rootCmd.PersistentFlags().StringVar(&cfg.SSHPort, "ssh-port", "", "SSH port")
	rootCmd.PersistentFlags().StringVar(&cfg.SSHKey, "ssh-key", "", "path to SSH key")

	// Image options
	rootCmd.PersistentFlags().StringVar(&cfg.Image, "image", "", "Docker image name")
	rootCmd.PersistentFlags().StringVar(&cfg.Dockerfile, "dockerfile", "", "path to Dockerfile")
	rootCmd.PersistentFlags().StringVar(&cfg.Tag, "tag", "", "Docker image tag")
	rootCmd.PersistentFlags().StringVar(&cfg.Platform, "platform", "", "Docker platform")
	rootCmd.PersistentFlags().StringSliceVar(&sliceOpts.buildArgs, "build-arg", nil, "build argument KEY=VALUE")

	// Container options
	rootCmd.PersistentFlags().StringVar(&cfg.ContainerName, "container-name", "", "name for the container")
	rootCmd.PersistentFlags().StringVar(&cfg.ContainerPort, "container-port", "", "container port")
	rootCmd.PersistentFlags().StringVar(&cfg.HostPort, "host-port", "", "host port")
	rootCmd.PersistentFlags().StringVar(&cfg.EnvFile, "env-file", "", "environment file")
	rootCmd.PersistentFlags().StringSliceVarP(&sliceOpts.volumes, "volume", "v", nil, "volume mount host:container")
	rootCmd.PersistentFlags().StringSliceVar(&sliceOpts.remoteCommands, "remote-command", nil, "remote command to execute after deployment")
	rootCmd.PersistentFlags().StringVar(&cfg.Network, "network", "", "Docker network to connect to")

	// Resource limits
	rootCmd.PersistentFlags().StringVar(&cfg.CPUs, "cpus", "", "number of CPUs")
	rootCmd.PersistentFlags().StringVar(&cfg.Memory, "memory", "", "memory limit")

	// Health check
	rootCmd.PersistentFlags().StringVar(&cfg.HealthCmd, "health-cmd", "", "health check command")
	rootCmd.PersistentFlags().StringVar(&cfg.HealthInterval, "health-interval", "", "health check interval")
	rootCmd.PersistentFlags().StringVar(&cfg.HealthTimeout, "health-timeout", "", "health check timeout")
	rootCmd.PersistentFlags().IntVar(&cfg.HealthRetries, "health-retries", 0, "health check retries")
	rootCmd.PersistentFlags().StringVar(&cfg.HealthStart, "health-start-period", "", "health check start period")

	// Container runtime
	rootCmd.PersistentFlags().StringVar(&cfg.RestartPolicy, "restart", "", "restart policy")
	rootCmd.PersistentFlags().StringSliceVarP(&sliceOpts.labels, "label", "l", nil, "container label KEY=VALUE")
	rootCmd.PersistentFlags().StringSliceVarP(&sliceOpts.envVars, "env", "e", nil, "environment variable KEY=VALUE")
	rootCmd.PersistentFlags().StringVar(&cfg.Entrypoint, "entrypoint", "", "override container entrypoint")
	rootCmd.PersistentFlags().StringVar(&cfg.Command, "command", "", "override container command")
	rootCmd.PersistentFlags().StringVar(&cfg.ContainerUser, "container-user", "", "user to run container as")
	rootCmd.PersistentFlags().StringVar(&cfg.Workdir, "workdir", "", "working directory inside container")
	rootCmd.PersistentFlags().StringVar(&cfg.Hostname, "hostname", "", "container hostname")
	rootCmd.PersistentFlags().StringSliceVar(&sliceOpts.extraHosts, "add-host", nil, "add host:ip mapping")

	// Security
	rootCmd.PersistentFlags().BoolVar(&cfg.Privileged, "privileged", false, "run container in privileged mode")
	rootCmd.PersistentFlags().BoolVar(&cfg.Init, "init", false, "run init inside container")
	rootCmd.PersistentFlags().BoolVar(&cfg.ReadOnly, "read-only", false, "mount root filesystem as read-only")
	rootCmd.PersistentFlags().StringSliceVar(&sliceOpts.capAdd, "cap-add", nil, "add Linux capability")
	rootCmd.PersistentFlags().StringSliceVar(&sliceOpts.capDrop, "cap-drop", nil, "drop Linux capability")

	// Storage
	rootCmd.PersistentFlags().StringSliceVar(&sliceOpts.tmpfs, "tmpfs", nil, "mount tmpfs")

	// Logging
	rootCmd.PersistentFlags().StringVar(&cfg.LogDriver, "log-driver", "", "logging driver")
	rootCmd.PersistentFlags().StringSliceVar(&sliceOpts.logOpts, "log-opt", nil, "log driver option KEY=VALUE")
	rootCmd.PersistentFlags().StringVar(&cfg.LogFile, "log-file", "", "path to log file")

	// Execution
	rootCmd.PersistentFlags().BoolVar(&cfg.DryRun, "dry-run", false, "preview deployment without executing")
	rootCmd.PersistentFlags().BoolVar(&cfg.Verbose, "verbose", false, "show detailed output")
	rootCmd.PersistentFlags().BoolVar(&cfg.JSONOutput, "json", false, "output stats as JSON")

	// Add subcommands
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(versionCmd)
}

func initConfig() error {
	// Initialize maps
	cfg.BuildArgs = make(map[string]string)
	cfg.Labels = make(map[string]string)
	cfg.Env = make(map[string]string)
	cfg.LogOpts = make(map[string]string)

	// Load config file
	fileConfig := loadConfigFile(cfgFile)

	// Merge file config with CLI flags
	cfg = mergeConfigs(fileConfig, cfg)

	// Parse slice flags into maps/slices
	parseSliceFlags()

	// Expand home path
	cfg.SSHKey = expandHomePath(cfg.SSHKey)

	// Generate versioned tag
	if cfg.Tag == "latest" {
		cfg.Tag = fmt.Sprintf("latest-%s", time.Now().Format("20060102150405"))
	}

	return nil
}

func parseSliceFlags() {
	for _, arg := range sliceOpts.buildArgs {
		if k, v, ok := parseKeyValue(arg); ok {
			cfg.BuildArgs[k] = v
		}
	}
	for _, arg := range sliceOpts.envVars {
		if k, v, ok := parseKeyValue(arg); ok {
			cfg.Env[k] = v
		}
	}
	for _, arg := range sliceOpts.labels {
		if k, v, ok := parseKeyValue(arg); ok {
			cfg.Labels[k] = v
		}
	}
	for _, arg := range sliceOpts.logOpts {
		if k, v, ok := parseKeyValue(arg); ok {
			cfg.LogOpts[k] = v
		}
	}

	if len(sliceOpts.volumes) > 0 {
		cfg.Volumes = sliceOpts.volumes
	}
	if len(sliceOpts.remoteCommands) > 0 {
		cfg.RemoteCommands = sliceOpts.remoteCommands
	}
	if len(sliceOpts.extraHosts) > 0 {
		cfg.ExtraHosts = sliceOpts.extraHosts
	}
	if len(sliceOpts.capAdd) > 0 {
		cfg.CapAdd = sliceOpts.capAdd
	}
	if len(sliceOpts.capDrop) > 0 {
		cfg.CapDrop = sliceOpts.capDrop
	}
	if len(sliceOpts.tmpfs) > 0 {
		cfg.Tmpfs = sliceOpts.tmpfs
	}
}

func parseKeyValue(s string) (key, value string, ok bool) {
	parts := strings.SplitN(s, "=", 2)
	if len(parts) == 2 {
		return parts[0], parts[1], true
	}
	return "", "", false
}

func expandHomePath(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[2:])
}

func runDeploy() error {
	log, err := logger.New(cfg.LogFile, cfg.Verbose)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer log.Close()

	if err := deploy.Deploy(&cfg, log); err != nil {
		log.Error("Deployment failed", err)
		return err
	}
	return nil
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show container stats from remote host",
	Long:  "Display CPU, memory, network, and other statistics for the deployed container.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.Host == "" || cfg.User == "" {
			return fmt.Errorf("host and user are required for stats")
		}

		log, err := logger.New(cfg.LogFile, cfg.Verbose)
		if err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}
		defer log.Close()

		stats, err := container.GetStats(&cfg, log)
		if err != nil {
			return err
		}

		if cfg.JSONOutput {
			jsonOutput, err := stats.ToJSON()
			if err != nil {
				return fmt.Errorf("failed to generate JSON output: %w", err)
			}
			fmt.Println(jsonOutput)
		} else {
			stats.PrintStats()
		}
		return nil
	},
}

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback to previous version",
	Long:  "Rollback the deployed container to its previous version.",
	RunE: func(cmd *cobra.Command, args []string) error {
		log, err := logger.New(cfg.LogFile, cfg.Verbose)
		if err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}
		defer log.Close()

		if err := deploy.Rollback(&cfg, log); err != nil {
			log.Error("Rollback failed", err)
			return err
		}
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("pipe version %s\n", Version)
	},
}
