# Configuration Reference

Complete reference for all Pipe configuration options.

## Overview

Pipe can be configured three ways (in order of priority):

1. **Command line flags** — Highest priority, overrides everything
2. **Environment variables** — Good for CI/CD and secrets
3. **Config file** (`pipe.yaml`) — Best for project defaults

```bash
# These all work together
export HOST=server.com           # Environment variable
pipe --tag v1.0.0                # CLI flag overrides config file
# pipe.yaml provides defaults for everything else
```

## Config File

Pipe automatically looks for `pipe.yaml` or `pipe.yml` in your current directory.

```bash
# Use default config file
pipe

# Use specific config file
pipe --config production.yaml
```

### Minimal Config

```yaml
# pipe.yaml
host: server.com
user: deploy
image: my-app
containerPort: "3000"
hostPort: "3000"
```

### Full Config Example

```yaml
# pipe.yaml - Complete reference

# ═══════════════════════════════════════════════════════════════
# CONNECTION
# ═══════════════════════════════════════════════════════════════
host: server.com              # Required: Remote host
user: deploy                  # SSH user (default: current user)
sshPort: "22"                 # SSH port
sshKey: ~/.ssh/id_rsa         # Path to SSH private key

# ═══════════════════════════════════════════════════════════════
# IMAGE
# ═══════════════════════════════════════════════════════════════
image: my-app                 # Docker image name
dockerfile: Dockerfile        # Dockerfile path
tag: latest                   # Image tag (use ${VAR} for dynamic)
platform: linux/amd64         # Target platform

buildArgs:                    # Build-time variables
  VERSION: "1.0.0"
  GIT_SHA: ${GIT_SHA}         # Expanded from environment

# ═══════════════════════════════════════════════════════════════
# CONTAINER
# ═══════════════════════════════════════════════════════════════
containerName: my-app         # Container name
containerPort: "3000"         # Port inside container
hostPort: "3000"              # Port on host
restartPolicy: unless-stopped # no, always, on-failure, unless-stopped

# Networking
network: my-network           # Docker network name
hostname: my-app              # Container hostname
extraHosts:                   # /etc/hosts entries
  - "api.internal:10.0.0.5"

# Environment
envFile: .env.production      # Load from file
env:                          # Inline variables
  NODE_ENV: production
  LOG_LEVEL: info

# Volumes
volumes:
  - /host/data:/app/data      # Persistent storage
  - /host/config:/app/config:ro  # Read-only

# ═══════════════════════════════════════════════════════════════
# RESOURCES
# ═══════════════════════════════════════════════════════════════
cpus: "0.5"                   # CPU limit (0.5 = half a core)
memory: "512m"                # Memory limit (m=MB, g=GB)

# ═══════════════════════════════════════════════════════════════
# HEALTH CHECK
# ═══════════════════════════════════════════════════════════════
healthCmd: "curl -f http://localhost:3000/health || exit 1"
healthInterval: "30s"         # Time between checks
healthTimeout: "10s"          # Timeout per check
healthRetries: 3              # Failures before unhealthy
healthStartPeriod: "60s"      # Grace period for startup

# ═══════════════════════════════════════════════════════════════
# SECURITY
# ═══════════════════════════════════════════════════════════════
containerUser: "1000:1000"    # Run as specific user:group
privileged: false             # Full host access (dangerous!)
readOnly: false               # Read-only root filesystem
init: true                    # Use init process (recommended)

capAdd:                       # Add Linux capabilities
  - NET_BIND_SERVICE
capDrop:                      # Remove capabilities
  - ALL                       # Drop all, then add specific ones

# ═══════════════════════════════════════════════════════════════
# STORAGE
# ═══════════════════════════════════════════════════════════════
tmpfs:                        # In-memory filesystems
  - /tmp
  - /var/run

# ═══════════════════════════════════════════════════════════════
# LOGGING
# ═══════════════════════════════════════════════════════════════
logDriver: json-file          # json-file, syslog, none, etc.
logOpts:
  max-size: "10m"             # Rotate at 10MB
  max-file: "3"               # Keep 3 files

# ═══════════════════════════════════════════════════════════════
# METADATA
# ═══════════════════════════════════════════════════════════════
labels:
  app: my-app
  environment: production
  team: platform

# ═══════════════════════════════════════════════════════════════
# ADVANCED
# ═══════════════════════════════════════════════════════════════
entrypoint: ""                # Override ENTRYPOINT
command: ""                   # Override CMD
workdir: /app                 # Working directory

# ═══════════════════════════════════════════════════════════════
# POST-DEPLOYMENT
# ═══════════════════════════════════════════════════════════════
remoteCommands:
  - "docker system prune -f --filter 'until=24h'"
  - "echo 'Deployed!' >> /var/log/deploys.log"
```

---

## CLI Reference

### Core Options

| Flag | Env Variable | Default | Description |
|------|--------------|---------|-------------|
| `--config` | — | `pipe.yaml` | Config file path |
| `--host` | `HOST` | — | **Required.** Remote host |
| `--user` | `HOST_USER` | current user | SSH username |
| `--ssh-port` | `SSH_PORT` | `22` | SSH port |
| `--ssh-key` | `SSH_KEY_PATH` | — | SSH private key path |
| `--dry-run` | `DRY_RUN` | `false` | Preview without changes |
| `--verbose`, `-v` | `VERBOSE` | `false` | Detailed output |

### Image Options

| Flag | Env Variable | Default | Description |
|------|--------------|---------|-------------|
| `--image` | `DOCKER_IMAGE_NAME` | `app` | Image name |
| `--dockerfile` | `DOCKERFILE` | `Dockerfile` | Dockerfile path |
| `--tag` | `DOCKER_IMAGE_TAG` | `latest` | Image tag |
| `--platform` | `HOST_PLATFORM` | `linux/amd64` | Target platform |
| `--build-arg` | `DOCKER_BUILD_ARGS` | — | Build argument (repeatable) |

### Container Options

| Flag | Env Variable | Default | Description |
|------|--------------|---------|-------------|
| `--container-name` | `DOCKER_CONTAINER_NAME` | `app` | Container name |
| `--container-port` | `DOCKER_CONTAINER_PORT` | `3000` | Container port |
| `--host-port` | `HOST_PORT` | `3000` | Host port |
| `--restart` | `RESTART_POLICY` | `unless-stopped` | Restart policy |
| `--network` | `DOCKER_NETWORK` | — | Docker network |
| `--env-file` | `DOCKER_CONTAINER_ENV_FILE` | — | Environment file |
| `--env` | — | — | Environment variable (repeatable) |
| `--volume` | `DOCKER_VOLUMES` | — | Volume mount (repeatable) |

### Resource Limits

| Flag | Env Variable | Default | Description |
|------|--------------|---------|-------------|
| `--cpus` | `DOCKER_CPUS` | — | CPU limit (e.g., `0.5`, `2`) |
| `--memory` | `DOCKER_MEMORY` | — | Memory limit (e.g., `512m`, `2g`) |

### Health Check

| Flag | Env Variable | Default | Description |
|------|--------------|---------|-------------|
| `--health-cmd` | `HEALTH_CMD` | — | Health check command |
| `--health-interval` | `HEALTH_INTERVAL` | — | Check interval (e.g., `30s`) |
| `--health-timeout` | `HEALTH_TIMEOUT` | — | Check timeout (e.g., `10s`) |
| `--health-retries` | `HEALTH_RETRIES` | — | Failure threshold |
| `--health-start-period` | `HEALTH_START_PERIOD` | — | Startup grace period |

### Security

| Flag | Env Variable | Default | Description |
|------|--------------|---------|-------------|
| `--container-user` | `CONTAINER_USER` | — | User to run as |
| `--privileged` | `PRIVILEGED` | `false` | Privileged mode |
| `--read-only` | `READ_ONLY` | `false` | Read-only root fs |
| `--init` | `INIT` | `false` | Use init process |
| `--cap-add` | — | — | Add capability (repeatable) |
| `--cap-drop` | — | — | Drop capability (repeatable) |

### Advanced

| Flag | Env Variable | Default | Description |
|------|--------------|---------|-------------|
| `--hostname` | `CONTAINER_HOSTNAME` | — | Container hostname |
| `--workdir` | `WORKDIR` | — | Working directory |
| `--entrypoint` | — | — | Override entrypoint |
| `--command` | — | — | Override command |
| `--add-host` | — | — | Host mapping (repeatable) |
| `--label` | — | — | Label (repeatable) |
| `--tmpfs` | — | — | tmpfs mount (repeatable) |
| `--log-driver` | `LOG_DRIVER` | — | Logging driver |
| `--log-opt` | — | — | Log option (repeatable) |

### Execution

| Flag | Env Variable | Default | Description |
|------|--------------|---------|-------------|
| `--remote-command` | `REMOTE_COMMANDS` | — | Post-deploy command (repeatable) |
| `--rollback` | — | `false` | Rollback to previous |

---

## Environment Variable Expansion

Config files support `${VAR}` syntax for dynamic values:

```yaml
tag: ${VERSION:-latest}        # Use $VERSION or "latest" as fallback
buildArgs:
  GIT_SHA: ${GIT_SHA}          # Required - fails if not set
  BUILD_DATE: ${BUILD_DATE:-}  # Optional - empty if not set
```

Deploy with:

```bash
VERSION=v1.0.0 GIT_SHA=$(git rev-parse --short HEAD) pipe
```

---

## Directory Structure

Recommended project layout:

```
my-app/
├── Dockerfile              # Required
├── pipe.yaml               # Default config
├── pipe.production.yaml    # Production overrides (optional)
├── .env.production         # Environment variables (optional)
├── src/                    # Your application
└── .dockerignore           # Files to exclude from build
```

---

## Best Practices

### Use Config Files for Defaults

```yaml
# pipe.yaml - Team defaults
image: my-app
containerPort: "3000"
hostPort: "3000"
restartPolicy: unless-stopped
```

### Use Environment Variables for Secrets

```bash
# CI/CD pipeline
export HOST=$DEPLOY_SERVER
export SSH_KEY_PATH=/path/to/key
pipe --tag $CI_COMMIT_TAG
```

### Use CLI Flags for Overrides

```bash
# Quick one-off change
pipe --tag hotfix-123 --verbose
```

### Validate Before Deploy

```bash
# Always preview first in production
pipe --dry-run
```
