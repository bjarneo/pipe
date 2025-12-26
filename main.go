package main

import (
	"fmt"
	"os"

	"github.com/bjarneo/pipe/internal/config"
	"github.com/bjarneo/pipe/internal/container"
	"github.com/bjarneo/pipe/internal/deploy"
	"github.com/bjarneo/pipe/internal/logger"
)

// defaultLogFile is the default filename for deployment logs
const defaultLogFile = "deploy.log"

func main() {
	cfg := config.Load()

	log := initLogger(cfg.Verbose)
	defer log.Close()

	if cfg.ShowStats {
		if err := showContainerStats(&cfg, log); err != nil {
			log.Error("Failed to get stats", err)
			os.Exit(1)
		}
	} else if cfg.Rollback {
		if err := deploy.Rollback(&cfg, log); err != nil {
			log.Error("Rollback failed", err)
			os.Exit(1)
		}
	} else {
		if err := deploy.Deploy(&cfg, log); err != nil {
			log.Error("Deployment failed", err)
			os.Exit(1)
		}
	}
}

func initLogger(verbose bool) *logger.Logger {
	log, err := logger.New(defaultLogFile, verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	return log
}

func showContainerStats(cfg *config.Config, log *logger.Logger) error {
	// Minimal validation for stats - just need host, user, and container name
	if cfg.Host == "" || cfg.User == "" {
		return fmt.Errorf("host and user are required for --stats")
	}

	stats, err := container.GetStats(cfg, log)
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
}
