# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with this repository.

## Project Overview

Pipe is a Docker deployment CLI tool written in Go that transfers Docker images to remote hosts via SSH without requiring a registry. It uses delta transfers to only send changed layers.

## Architecture

```
pipe/
├── main.go                 # Entry point - loads config, runs deploy, rollback, or stats
├── internal/
│   ├── config/            # Configuration loading (CLI, env vars, YAML file)
│   ├── container/         # Container stats retrieval and display
│   ├── deploy/            # Deployment and rollback orchestration
│   ├── docker/            # Docker build, transfer, and container management
│   ├── ssh/               # SSH command execution
│   ├── logger/            # Logging utility
│   ├── format/            # Output formatting utilities
│   └── stats/             # Deployment statistics
├── docs/                   # Documentation
└── scripts/                # Git hooks and utilities
```

## Key Commands

```bash
# Build the project
make build

# Run all checks (format, lint, test, build)
make

# Format code
make fmt

# Run linter
make lint

# Run tests
make test

# Install git pre-commit hooks
make install-hooks
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

## Versioned Tags

When using the default `latest` tag, pipe automatically adds a timestamp suffix (e.g., `latest-20251226150123`) to enable rollback functionality by preserving multiple versions on the remote host.

## Key Files to Understand

- `internal/config/config.go` - All configuration options, validation, and loading logic
- `internal/container/container.go` - Remote container stats via docker stats/inspect
- `internal/docker/docker.go` - Docker build, delta transfer, and container deployment
- `internal/deploy/deploy.go` - Main deployment orchestration and dry-run
- `internal/ssh/ssh.go` - SSH/SCP command building and execution

## Adding New Features

When adding new container options:
1. Add field to `Config` struct in `config.go` with json/yaml tags
2. Add CLI flag in `defineFlags()` function
3. Add to `mergeConfig()` with appropriate env var mapping (for bool flags, copy directly like `result.ShowStats = cliConfig.ShowStats`)
4. Add to `expandEnvVars()` if string type
5. Add to `buildContainerConfig()` in `docker.go` if it affects container run
6. Update help text and documentation
7. Run `make check` to verify formatting, linting, and tests pass

## Testing Locally

Without a real remote host, use `--dry-run` to preview:
```bash
./pipe --host example.com --user deploy --dry-run
```

To test stats without deploying:
```bash
./pipe --stats --host example.com --user deploy
```

## Code Quality

- **Linting**: Uses golangci-lint (config in `.golangci.yml`)
- **Formatting**: Uses gofmt
- **Pre-commit hooks**: Install with `make install-hooks`

See `docs/lint.md` for full details.

## Common Patterns

- Use `ssh.GetCommand(cfg)` for SSH commands
- Use `ssh.GetSCPCommand(cfg)` for SCP commands
- Use `ssh.ExecuteCommand(log, cmd, description)` for running commands
- Maps like `BuildArgs`, `Env`, `Labels` are KEY=VALUE pairs
- Array flags (volumes, caps, etc.) can be specified multiple times
- For SSH commands with docker format templates, use single quotes outside and double quotes for the format string: `ssh user@host 'docker inspect --format "{{.Id}}" container'`
- Use constants for strings repeated 3+ times (enforced by goconst linter)
