# Examples

Practical examples for common deployment scenarios.

> **Tip:** For complete, runnable examples, see the [examples/](../examples/) directory.

## Basic Deployments

### Minimal Command Line

```bash
pipe --host server.com --user deploy
```

### With Custom Ports

```bash
pipe --host server.com --user deploy \
  --container-port 8080 \
  --host-port 80
```

### With Environment File

```bash
pipe --host server.com --user deploy \
  --env-file .env.production
```

### Dry Run (Preview)

See what would happen without making changes:

```bash
pipe --dry-run
```

## Configuration File Examples

### Minimal Config

```yaml
# pipe.yaml
host: server.com
user: deploy
image: my-app
containerPort: "3000"
hostPort: "3000"
```

### Production-Ready Config

```yaml
# pipe.yaml
host: prod.example.com
user: deploy
image: my-app
tag: ${VERSION:-latest}

containerName: my-app
containerPort: "3000"
hostPort: "80"

# Always restart unless manually stopped
restartPolicy: unless-stopped

# Health monitoring
healthCmd: "curl -f http://localhost:3000/health || exit 1"
healthInterval: "30s"
healthTimeout: "10s"
healthRetries: 3

# Resource limits
cpus: "1"
memory: "512m"

# Environment
env:
  NODE_ENV: production
  LOG_LEVEL: warn

# Metadata
labels:
  app: my-app
  environment: production
  managed-by: pipe
```

### Secure Container Config

```yaml
# pipe.yaml - Security-hardened
host: server.com
user: deploy
image: my-app

# Run as non-root
containerUser: "1000:1000"

# Read-only filesystem
readOnly: true
tmpfs:
  - /tmp
  - /var/run

# Drop unnecessary capabilities
capDrop:
  - ALL
capAdd:
  - NET_BIND_SERVICE

# Use init process
init: true
```

## Build Arguments

### Single Argument

```bash
pipe --build-arg VERSION=1.0.0
```

### Multiple Arguments

```bash
pipe --build-arg VERSION=1.0.0 \
     --build-arg GIT_SHA=$(git rev-parse --short HEAD) \
     --build-arg BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
```

### In Config File

```yaml
buildArgs:
  VERSION: ${VERSION:-dev}
  GIT_SHA: ${GIT_SHA}
  BUILD_DATE: ${BUILD_DATE}
```

Deploy with:

```bash
VERSION=1.0.0 GIT_SHA=$(git rev-parse --short HEAD) pipe
```

## Environment Variables

### Inline Variables

```bash
pipe --env NODE_ENV=production --env LOG_LEVEL=info
```

### From File

```bash
pipe --env-file .env.production
```

### Mixed (File + Overrides)

```yaml
# pipe.yaml
envFile: .env.production
env:
  LOG_LEVEL: debug  # Override for debugging
```

## Volume Mounts

### Single Volume

```bash
pipe --volume /data:/app/data
```

### Multiple Volumes

```bash
pipe --volume /data:/app/data \
     --volume /logs:/app/logs \
     --volume /config:/app/config:ro
```

### In Config

```yaml
volumes:
  - /data:/app/data
  - /logs:/app/logs
  - /config:/app/config:ro  # read-only
```

## Networking

### Custom Network

```bash
pipe --network my-network
```

### With Extra Hosts

```yaml
network: my-network
extraHosts:
  - "api.internal:192.168.1.100"
  - "db.internal:192.168.1.101"
```

## Resource Limits

```yaml
# Limit CPU and memory
cpus: "0.5"      # Half a CPU
memory: "256m"   # 256 MB RAM
```

## Logging Configuration

### JSON File with Rotation

```yaml
logDriver: json-file
logOpts:
  max-size: "10m"
  max-file: "3"
```

### Syslog

```yaml
logDriver: syslog
logOpts:
  syslog-address: "udp://logs.example.com:514"
  tag: "my-app"
```

## Multi-Environment Setup

### Directory Structure

```
my-app/
├── Dockerfile
├── pipe.yaml           # Default/development
├── pipe.staging.yaml   # Staging overrides
└── pipe.production.yaml # Production overrides
```

### Usage

```bash
# Development
pipe

# Staging
pipe --config pipe.staging.yaml

# Production
pipe --config pipe.production.yaml
```

## Rollback

> **Important:** Rollback requires different tags for each deployment. Don't overwrite the same tag.

### Deploy with Version Tags

```bash
# Deploy v1.0.0
VERSION=v1.0.0 pipe --tag v1.0.0

# Deploy v1.0.1
VERSION=v1.0.1 pipe --tag v1.0.1

# Oops! Rollback to v1.0.0
pipe --rollback
```

### Automatic Versioning

```bash
# Use git tag or commit SHA
pipe --tag $(git describe --tags --always)
```

## Post-Deployment Commands

Run commands on the remote server after deployment:

```yaml
remoteCommands:
  - "docker system prune -f --filter 'until=24h'"
  - "echo 'Deployed at $(date)' >> /var/log/deployments.log"
```

Or via CLI:

```bash
pipe --remote-command "docker system prune -f"
```

## Complete Real-World Example

```yaml
# pipe.yaml - Full production setup
host: ${DEPLOY_HOST}
user: deploy
sshKey: ~/.ssh/deploy_key

image: my-company/my-app
tag: ${CI_COMMIT_TAG:-latest}
platform: linux/amd64

containerName: my-app
containerPort: "3000"
hostPort: "443"
restartPolicy: unless-stopped

network: production

healthCmd: "wget -q --spider http://localhost:3000/health || exit 1"
healthInterval: "30s"
healthTimeout: "10s"
healthRetries: 3
healthStartPeriod: "60s"

cpus: "2"
memory: "1g"

envFile: .env.production
env:
  NODE_ENV: production

volumes:
  - /var/data/my-app:/app/data
  - /var/log/my-app:/app/logs

labels:
  app: my-app
  version: ${CI_COMMIT_TAG:-latest}
  environment: production

buildArgs:
  VERSION: ${CI_COMMIT_TAG:-dev}
  BUILD_DATE: ${CI_PIPELINE_CREATED_AT}
  GIT_SHA: ${CI_COMMIT_SHA}

logDriver: json-file
logOpts:
  max-size: "50m"
  max-file: "5"

init: true
readOnly: false

remoteCommands:
  - "docker system prune -f --filter 'until=72h'"
```
