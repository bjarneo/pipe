# GitHub Actions

Use the Pipe GitHub Action to deploy Docker containers directly from your CI/CD workflows.

## Quick Start

```yaml
- name: Deploy to production
  uses: bjarneo/pipe@main
  with:
    host: ${{ secrets.DEPLOY_HOST }}
    user: deploy
    ssh_key: ${{ secrets.SSH_PRIVATE_KEY }}
    image: my-app
    container_name: my-app
    container_port: 3000
    host_port: 3000
```

## Action Inputs

### Core Settings

| Input     | Required | Default | Description |
|-----------|----------|---------|-------------|
| host      | Yes      |         | Remote host to deploy to |
| user      | No       | root    | SSH user for remote host |
| ssh_key   | Yes      |         | SSH private key content |
| ssh_port  | No       | 22      | SSH port |

### Image Settings

| Input      | Required | Default     | Description |
|------------|----------|-------------|-------------|
| image      | Yes      |             | Docker image name |
| dockerfile | No       | Dockerfile  | Path to Dockerfile |
| tag        | No       | latest      | Docker image tag |
| platform   | No       | linux/amd64 | Docker platform |
| build_args | No       |             | Build arguments (comma-separated KEY=VALUE) |

### Container Settings

| Input          | Required | Default        | Description |
|----------------|----------|----------------|-------------|
| container_name | Yes      |                | Name for the container |
| container_port | Yes      |                | Container port |
| host_port      | Yes      |                | Host port |
| env_file       | No       |                | Environment file path |
| env_vars       | No       |                | Environment variables (comma-separated KEY=VALUE) |
| network        | No       |                | Docker network to connect to |
| volumes        | No       |                | Volume mounts (comma-separated host:container) |
| restart_policy | No       | unless-stopped | Restart policy |

### Resource Limits

| Input  | Required | Default | Description |
|--------|----------|---------|-------------|
| cpus   | No       |         | Number of CPUs (e.g., "0.5") |
| memory | No       |         | Memory limit (e.g., "512m") |

### Health Check

| Input           | Required | Default | Description |
|-----------------|----------|---------|-------------|
| health_cmd      | No       |         | Health check command |
| health_interval | No       |         | Time between checks (e.g., "30s") |
| health_timeout  | No       |         | Check timeout (e.g., "10s") |
| health_retries  | No       |         | Consecutive failures needed |

### Advanced Container Options

| Input          | Required | Default | Description |
|----------------|----------|---------|-------------|
| labels         | No       |         | Container labels (comma-separated KEY=VALUE) |
| entrypoint     | No       |         | Override container entrypoint |
| command        | No       |         | Override container command |
| container_user | No       |         | User to run container as |
| workdir        | No       |         | Working directory inside container |
| hostname       | No       |         | Container hostname |

### Security

| Input      | Required | Default | Description |
|------------|----------|---------|-------------|
| privileged | No       | false   | Run in privileged mode |
| read_only  | No       | false   | Read-only root filesystem |
| init       | No       | false   | Run init inside container |

### Logging

| Input      | Required | Default | Description |
|------------|----------|---------|-------------|
| log_driver | No       |         | Logging driver |
| log_opts   | No       |         | Log driver options (comma-separated KEY=VALUE) |

### Execution Options

| Input           | Required | Default | Description |
|-----------------|----------|---------|-------------|
| remote_commands | No       |         | Post-deployment commands (comma-separated) |
| rollback        | No       | false   | Rollback to previous version |
| verbose         | No       | false   | Show detailed output |

## Examples

### Basic Deployment

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

      - name: Deploy to production
        uses: bjarneo/pipe@main
        with:
          host: ${{ secrets.DEPLOY_HOST }}
          user: deploy
          ssh_key: ${{ secrets.SSH_PRIVATE_KEY }}
          image: my-app
          container_name: my-app
          container_port: 3000
          host_port: 3000
```

### Deployment with Version Tags

```yaml
name: Deploy Release

on:
  push:
    tags:
      - 'v*'

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Get version from tag
        id: version
        run: echo "VERSION=${GITHUB_REF#refs/tags/}" >> $GITHUB_OUTPUT

      - name: Deploy to production
        uses: bjarneo/pipe@main
        with:
          host: ${{ secrets.DEPLOY_HOST }}
          user: deploy
          ssh_key: ${{ secrets.SSH_PRIVATE_KEY }}
          image: my-app
          tag: ${{ steps.version.outputs.VERSION }}
          container_name: my-app
          container_port: 3000
          host_port: 80
          env_file: .env.production
          build_args: VERSION=${{ steps.version.outputs.VERSION }},GIT_SHA=${{ github.sha }}
```

### Deployment with Health Check

```yaml
- name: Deploy with health check
  uses: bjarneo/pipe@main
  with:
    host: ${{ secrets.DEPLOY_HOST }}
    user: deploy
    ssh_key: ${{ secrets.SSH_PRIVATE_KEY }}
    image: my-app
    container_name: my-app
    container_port: 3000
    host_port: 3000
    health_cmd: "curl -f http://localhost:3000/health || exit 1"
    health_interval: "30s"
    health_timeout: "10s"
    health_retries: "3"
```

### Deployment with Resource Limits

```yaml
- name: Deploy with limits
  uses: bjarneo/pipe@main
  with:
    host: ${{ secrets.DEPLOY_HOST }}
    user: deploy
    ssh_key: ${{ secrets.SSH_PRIVATE_KEY }}
    image: my-app
    container_name: my-app
    container_port: 3000
    host_port: 3000
    cpus: "0.5"
    memory: "512m"
```

### Rollback Workflow

```yaml
name: Rollback

on:
  workflow_dispatch:
    inputs:
      reason:
        description: 'Reason for rollback'
        required: true
        type: string

jobs:
  rollback:
    runs-on: ubuntu-latest
    steps:
      - name: Rollback to previous version
        uses: bjarneo/pipe@main
        with:
          host: ${{ secrets.DEPLOY_HOST }}
          user: deploy
          ssh_key: ${{ secrets.SSH_PRIVATE_KEY }}
          image: my-app
          container_name: my-app
          container_port: 3000
          host_port: 3000
          rollback: true
```

### Multi-Environment Deployment

```yaml
name: Deploy

on:
  push:
    branches:
      - main
      - develop

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set environment
        id: env
        run: |
          if [ "${{ github.ref }}" = "refs/heads/main" ]; then
            echo "ENV=production" >> $GITHUB_OUTPUT
            echo "HOST=${{ secrets.PROD_HOST }}" >> $GITHUB_OUTPUT
            echo "PORT=80" >> $GITHUB_OUTPUT
          else
            echo "ENV=staging" >> $GITHUB_OUTPUT
            echo "HOST=${{ secrets.STAGING_HOST }}" >> $GITHUB_OUTPUT
            echo "PORT=8080" >> $GITHUB_OUTPUT
          fi

      - name: Deploy
        uses: bjarneo/pipe@main
        with:
          host: ${{ steps.env.outputs.HOST }}
          user: deploy
          ssh_key: ${{ secrets.SSH_PRIVATE_KEY }}
          image: my-app
          tag: ${{ steps.env.outputs.ENV }}-${{ github.sha }}
          container_name: my-app-${{ steps.env.outputs.ENV }}
          container_port: 3000
          host_port: ${{ steps.env.outputs.PORT }}
          env_vars: NODE_ENV=${{ steps.env.outputs.ENV }}
          labels: environment=${{ steps.env.outputs.ENV }},commit=${{ github.sha }}
```

## Secrets Setup

Add these secrets to your GitHub repository (Settings > Secrets > Actions):

| Secret           | Description |
|------------------|-------------|
| DEPLOY_HOST      | Remote server hostname or IP |
| SSH_PRIVATE_KEY  | SSH private key for authentication |

### Generating SSH Keys

```bash
# Generate a new SSH key pair
ssh-keygen -t ed25519 -C "github-actions-deploy" -f deploy_key

# Add public key to remote server
ssh-copy-id -i deploy_key.pub user@host

# Copy private key content to GitHub secret
cat deploy_key
```

## Deployment Logs

The action automatically uploads deployment logs as an artifact. You can download them from the Actions tab in your repository for debugging.
