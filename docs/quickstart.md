# Quick Start

Deploy your first container with Pipe in under 5 minutes.

## Prerequisites

Before you start, make sure you have:

- **Docker** installed on your local machine
- **Docker** installed on your remote server
- **SSH access** to the remote server (key-based authentication)

**Quick check:**

```bash
# Local Docker
docker --version

# Remote Docker (replace with your server)
ssh user@your-server.com "docker --version"
```

## Step 1: Install Pipe

```bash
curl -fsSL https://raw.githubusercontent.com/bjarneo/pipe/main/install.sh | sh
```

## Step 2: Create a Simple App

Create a project directory with these two files:

**Dockerfile:**
```dockerfile
FROM nginx:alpine
COPY index.html /usr/share/nginx/html/
EXPOSE 80
```

**index.html:**
```html
<!DOCTYPE html>
<html>
<body>
  <h1>Hello from Pipe!</h1>
</body>
</html>
```

## Step 3: Deploy

```bash
pipe --host your-server.com --user deploy --container-port 80 --host-port 8080
```

```
[1/4] Building image... done
[2/4] Analyzing layers... 3 changed, 12 cached
[3/4] Transferring delta... 47MB (saved 312MB)
[4/4] Starting container... running
```

**That's it!** Visit `http://your-server.com:8080` to see your app.

## Step 4: Create a Config File (Optional)

For easier deployments, create a `pipe.yaml`:

```yaml
host: your-server.com
user: deploy
image: my-app
containerPort: "80"
hostPort: "8080"
```

Now deploy with just:

```bash
pipe
```

## What Happened?

Pipe automatically:

1. Built your Docker image locally
2. Transferred only the changed layers to your server
3. Stopped any existing container
4. Started your new container

## Common Commands

```bash
# Preview what would happen (no changes)
pipe --dry-run

# Deploy with a specific tag
pipe --tag v1.0.0

# See detailed output
pipe --verbose

# Rollback to previous version
pipe rollback

# Show container stats
pipe stats
```

## Adding Health Checks

Keep your container healthy with automatic checks:

```yaml
# pipe.yaml
host: your-server.com
user: deploy
image: my-app
containerPort: "3000"
hostPort: "3000"

healthCmd: "curl -f http://localhost:3000/health || exit 1"
healthInterval: "30s"
healthRetries: 3
```

## Using Environment Variables

Pass configuration to your container:

```yaml
# pipe.yaml
env:
  NODE_ENV: production
  LOG_LEVEL: info
  API_URL: https://api.example.com

# Or use a file
envFile: .env.production
```

## Next Steps

- [Configuration Reference](configuration.md) - All available options
- [Examples](../examples/) - Real-world project examples
- [GitHub Actions](github-actions.md) - CI/CD integration
- [Delta Transfer](delta-transfer.md) - How layer caching works

## Need Help?

### SSH connection failed

```bash
# Test your connection manually
ssh -v user@your-server.com "echo connected"

# Custom SSH port?
pipe --ssh-port 2222 --host your-server.com --user deploy
```

### Docker permission denied

```bash
# On the remote server, add user to docker group
sudo usermod -aG docker $USER
# Then log out and back in
```

### Container won't start

```bash
# Check container logs on remote
ssh user@your-server.com "docker logs my-app"

# Use verbose mode locally
pipe --verbose
```
