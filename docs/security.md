# Security Guide

This guide covers security best practices for using Pipe, including how to handle secrets, environment variables, and secure your deployments.

## Overview

Pipe is designed with security in mind:
- **No registry required** - Images transfer directly via SSH, never touch public infrastructure
- **Input validation** - All configuration values are validated against strict patterns
- **Shell injection prevention** - Dangerous characters are blocked in user inputs

---

## Handling Secrets

### Environment Variables (Recommended for CI/CD)

Pass secrets through environment variables to avoid storing them in files or version control:

```bash
# Set secrets in your CI/CD pipeline
export DB_PASSWORD="your-secret"
export API_KEY="your-api-key"

# Use inline --env flags
pipe --env DB_PASSWORD="$DB_PASSWORD" --env API_KEY="$API_KEY"
```

In GitHub Actions:

```yaml
- name: Deploy
  env:
    DB_PASSWORD: ${{ secrets.DB_PASSWORD }}
    API_KEY: ${{ secrets.API_KEY }}
  run: |
    pipe --env DB_PASSWORD="$DB_PASSWORD" --env API_KEY="$API_KEY"
```

### Environment Files

For multiple environment variables, use an env file:

```bash
# .env.production (DO NOT commit to git!)
DB_HOST=db.example.com
DB_PASSWORD=supersecret
API_KEY=sk-1234567890
REDIS_URL=redis://localhost:6379
```

Deploy with the env file:

```bash
pipe --env-file .env.production
```

Or in `pipe.yaml`:

```yaml
envFile: .env.production
```

**How it works:**
1. Pipe copies the env file to `~/<filename>` on the remote host via SCP
2. Docker runs the container with `--env-file ~/<filename>`
3. The env file remains on the server for subsequent deployments

### Combining Env File and Inline Variables

You can use both approaches together - inline variables override file values:

```yaml
# pipe.yaml
envFile: .env.production
env:
  LOG_LEVEL: debug  # Override for this deployment
```

```bash
# CLI takes highest priority
pipe --env LOG_LEVEL=trace
```

---

## Secure File Transfer

### SSH Key Authentication

Always use SSH keys instead of passwords:

```bash
# Generate a dedicated deployment key
ssh-keygen -t ed25519 -f ~/.ssh/deploy_key -C "deploy@myapp"

# Add to remote server
ssh-copy-id -i ~/.ssh/deploy_key.pub user@server.com

# Use in deployment
pipe --ssh-key ~/.ssh/deploy_key
```

In `pipe.yaml`:

```yaml
sshKey: ~/.ssh/deploy_key
```

### SSH Port

If you use a non-standard SSH port:

```yaml
sshPort: "2222"
```

---

## Container Security Options

### Run as Non-Root User

Never run containers as root in production:

```yaml
containerUser: "1000:1000"  # UID:GID
```

Or create a user in your Dockerfile:

```dockerfile
RUN adduser --disabled-password --gecos '' appuser
USER appuser
```

### Drop All Capabilities

Use the principle of least privilege:

```yaml
capDrop:
  - ALL           # Drop all capabilities first
capAdd:
  - NET_BIND_SERVICE  # Add only what you need
```

### Read-Only Root Filesystem

Prevent container modifications:

```yaml
readOnly: true
tmpfs:
  - /tmp
  - /var/run
```

### Use Init Process

Handle zombie processes and signals properly:

```yaml
init: true
```

### Complete Security Configuration Example

```yaml
# pipe.yaml - Production security hardened
containerUser: "1000:1000"
readOnly: true
init: true

capDrop:
  - ALL
capAdd:
  - NET_BIND_SERVICE  # Only if binding to ports < 1024

tmpfs:
  - /tmp
  - /var/run

healthCmd: "curl -f http://localhost:3000/health || exit 1"
healthInterval: "30s"
healthRetries: 3
```

---

## Input Validation

Pipe validates all inputs to prevent shell injection attacks:

### Validated Patterns

| Field | Allowed Pattern |
|-------|-----------------|
| `host` | Valid hostname or IP address |
| `user` | Unix username (lowercase, alphanumeric) |
| `image` | Lowercase alphanumeric with `.`, `_`, `/`, `-` |
| `tag` | Alphanumeric with `.`, `_`, `-` |
| `containerName` | Alphanumeric with `_`, `.`, `-` |
| `platform` | Whitelisted values only |

### Blocked Characters

The following characters are blocked in paths and values to prevent shell injection:

```
; & | $ ` \ \n \r " ' < > ( ) { }
```

### Path Security

- Path traversal (`..`) is blocked in file paths
- `envFile` and `dockerfile` must be relative paths
- Absolute paths are rejected for security-sensitive fields

---

## CI/CD Security Best Practices

### GitHub Actions

```yaml
name: Deploy
on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup SSH Key
        run: |
          mkdir -p ~/.ssh
          echo "${{ secrets.SSH_PRIVATE_KEY }}" > ~/.ssh/deploy_key
          chmod 600 ~/.ssh/deploy_key
          
      - name: Deploy
        env:
          HOST: ${{ vars.DEPLOY_HOST }}
          HOST_USER: ${{ vars.DEPLOY_USER }}
          DB_PASSWORD: ${{ secrets.DB_PASSWORD }}
        run: |
          pipe --ssh-key ~/.ssh/deploy_key --env DB_PASSWORD="$DB_PASSWORD"
```

### Never Commit Secrets

Add to `.gitignore`:

```
.env
.env.*
*.pem
*.key
deploy_key
```

### Use Environment Variables for Sensitive Config

```yaml
# pipe.yaml - Safe to commit
host: ${DEPLOY_HOST}
user: ${DEPLOY_USER}
sshKey: ${SSH_KEY_PATH}
```

---

## Network Security

### Use Docker Networks

Isolate containers on internal networks:

```yaml
network: internal-network
```

### Limit Port Exposure

Only expose necessary ports:

```yaml
containerPort: "3000"
hostPort: "3000"  # Consider using a reverse proxy
```

### Health Checks

Ensure your application is healthy:

```yaml
healthCmd: "curl -f http://localhost:3000/health || exit 1"
healthInterval: "30s"
healthTimeout: "10s"
healthRetries: 3
healthStartPeriod: "60s"
```

---

## Resource Limits

Prevent resource exhaustion attacks:

```yaml
cpus: "1"       # Limit to 1 CPU
memory: "512m"  # Limit to 512MB RAM
```

---

## Logging

### Secure Log Configuration

Limit log file size to prevent disk exhaustion:

```yaml
logDriver: json-file
logOpts:
  max-size: "10m"
  max-file: "3"
```

### Disable Logging for Sensitive Apps

```yaml
logDriver: none
```

---

## Remote Commands Security

Remote commands are validated for dangerous patterns. The following are blocked:

- `rm -rf /`
- `mkfs`
- `dd if=`
- `> /dev/`

Use remote commands only for safe post-deployment tasks:

```yaml
remoteCommands:
  - "docker system prune -f --filter 'until=24h'"
  - "echo 'Deployed at $(date)' >> /var/log/deploys.log"
```

---

## Security Checklist

Before deploying to production:

- [ ] SSH key authentication configured (no passwords)
- [ ] Secrets passed via environment variables, not config files
- [ ] `.env` files excluded from git
- [ ] Container runs as non-root user
- [ ] Capabilities dropped (at minimum `--cap-drop ALL`)
- [ ] Read-only filesystem enabled (with tmpfs for writeable paths)
- [ ] Resource limits set (CPU and memory)
- [ ] Health checks configured
- [ ] Log rotation configured
- [ ] Docker network isolation configured
- [ ] Dry-run tested before production deployment

```bash
# Always preview first
pipe --dry-run
```
