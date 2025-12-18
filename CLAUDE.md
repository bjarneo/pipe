# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with this repository.

## Project Overview

Pipe is a Docker deployment CLI tool written in Go that transfers Docker images to remote hosts via SSH without requiring a registry. It uses delta transfers to only send changed layers.

## Architecture

```
pipe/
├── main.go                 # Entry point - loads config, runs deploy or rollback
├── internal/
│   ├── config/            # Configuration loading (CLI, env vars, YAML file)
│   ├── deploy/            # Deployment and rollback orchestration
│   ├── docker/            # Docker build, transfer, and container management
│   ├── ssh/               # SSH command execution
│   ├── logger/            # Logging utility
│   └── stats/             # Deployment statistics
└── docs/                   # Documentation
```

## Key Commands

```bash
# Build the project
go build -o pipe .

# Run tests
go test ./...

# Run with version info
go build -ldflags "-X github.com/bjarneo/pipe/internal/config.version=1.0.0" -o pipe .
```

## Configuration Priority

Configuration is loaded in this order (highest to lowest priority):
1. CLI flags
2. Environment variables
3. Config file (pipe.yaml or pipe.yml)
4. Default values

## Core Flow

1. **Load Config** - Merge CLI flags, env vars, and YAML config
2. **Validate** - Check all required fields and validate patterns
3. **Build** - Build Docker image locally with specified platform
4. **Transfer** - Delta transfer only changed layers via SSH
5. **Deploy** - Stop old container, start new one with all options
6. **Post-deploy** - Execute remote commands if specified

## Key Files to Understand

- `internal/config/config.go` - All configuration options, validation, and loading logic
- `internal/docker/docker.go` - Docker build, delta transfer, and container deployment
- `internal/deploy/deploy.go` - Main deployment orchestration and dry-run
- `internal/ssh/ssh.go` - SSH/SCP command building and execution

## Adding New Features

When adding new container options:
1. Add field to `Config` struct in `config.go` with json/yaml tags
2. Add CLI flag in `Load()` function
3. Add to `mergeConfig()` with appropriate env var mapping
4. Add to `expandEnvVars()` if string type
5. Add to `buildContainerConfig()` in `docker.go`
6. Update help text and documentation

## Testing Locally

Without a real remote host, use `--dry-run` to preview:
```bash
./pipe --host example.com --user deploy --dry-run
```

## Common Patterns

- Use `ssh.GetCommand(cfg)` for SSH commands
- Use `ssh.GetSCPCommand(cfg)` for SCP commands
- Use `ssh.ExecuteCommand(log, cmd, description)` for running commands
- Maps like `BuildArgs`, `Env`, `Labels` are KEY=VALUE pairs
- Array flags (volumes, caps, etc.) can be specified multiple times
