package main

import (
	"fmt"
	"os"

	"github.com/bjarneo/pipe/internal/config"
	"github.com/bjarneo/pipe/internal/deploy"
	"github.com/bjarneo/pipe/internal/logger"
)

func main() {
	cfg := config.Load()

	log := initLogger(cfg.Verbose)
	defer log.Close()

	if cfg.Rollback {
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
	log, err := logger.New("deploy.log", verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	return log
}
