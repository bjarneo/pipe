<p align="center">
  <img src="logo.png" alt="Pipe" width="200">
</p>

<h1 align="center">Pipe</h1>

<p align="center">
  <strong>Deploy Docker containers anywhere via SSH - no registry required.</strong>
</p>

Pipe builds your Docker image locally, transfers only the changed layers to your server, and starts your container. It's like `git push` for containers.

## Why Not Use a Registry?

- **No Infrastructure Required** - No need to set up, authenticate, or pay for a registry (Docker Hub, ECR, Harbor, etc.)
- **Direct Path** - Data travels directly from your machine to the target server, no upload-then-download round trip
- **Security by Default** - Uses your existing SSH keys and encrypted tunnels, no registry credentials to manage
- **Air-Gapped Networks** - Works when the target server has no internet access but you have SSH access via a jump box

### When to Use What

**Use Pipe when:**
- Deploying to a single remote VPS for smaller projects
- The environment is locked down and cannot reach an external registry
- You want to avoid the "Registry Dance" (tagging, logging in, pushing, pulling) for a minor tweak
- You want to keep it simple, like a Capistrano deployment

**Use a Registry when:**
- Working in a team or production environment
- You have more than one target server
- You want CI/CD pipelines to handle deployments automatically
- Your image is large and you want layer caching across multiple servers

## Why Pipe?

- **No registry needed** - Deploy directly from your machine to any server
- **Fast transfers** - Only changed layers are sent (delta sync)
- **Simple config** - One YAML file, sensible defaults
- **Zero dependencies** - Single binary, works anywhere

## Quick Start

```bash
curl -fsSL https://raw.githubusercontent.com/bjarneo/pipe/main/install.sh | sh
```

```bash
pipe --host server.com --user deploy
```

```
[1/4] Building image... done
[2/4] Analyzing layers... 3 changed, 12 cached
[3/4] Transferring delta... 47MB (saved 312MB)
[4/4] Starting container... running
```

**[Full Quick Start Guide](docs/quickstart.md)**

## How It Works

```
Local Machine                     Remote Server
+-------------+                  +-------------+
|  Dockerfile | ---- SSH ---->   |   Docker    |
|  + Code     |   (delta only)   |  Container  |
+-------------+                  +-------------+
```

1. **Build** - Creates Docker image locally
2. **Compare** - Checks which layers exist on remote
3. **Transfer** - Sends only new/changed layers via SSH
4. **Deploy** - Stops old container, starts new one
5. **Cleanup** - Keeps last 5 releases for rollback

## Config File

Create `pipe.yaml` in your project:

```yaml
host: server.com
user: deploy
image: my-app
containerPort: "3000"
hostPort: "3000"

# Health monitoring
healthCmd: "curl -f http://localhost:3000/health || exit 1"
healthInterval: "30s"
healthRetries: 3

# Environment
env:
  NODE_ENV: production

# Build args (supports ${VAR} expansion)
buildArgs:
  VERSION: ${VERSION:-latest}
  GIT_SHA: ${GIT_SHA}
```

Then just run:

```bash
VERSION=1.0.0 GIT_SHA=$(git rev-parse --short HEAD) pipe
```

## Common Commands

```bash
pipe                          # Deploy using pipe.yaml
pipe --dry-run                # Preview without making changes
pipe --verbose                # Show detailed output
pipe --tag v2.0.0             # Deploy specific version
pipe rollback                 # Rollback to previous version
pipe stats                    # Show container stats (CPU, memory, network)
pipe stats --json             # Output stats as JSON
pipe --config prod.yaml       # Use different config
pipe version                  # Show version information
pipe help                     # Show help
pipe completion bash          # Generate shell completions
```

## Features

| Feature | Description |
|---------|-------------|
| **Delta transfers** | Only changed layers transfer - 10x faster after first deploy |
| **Health checks** | Monitor container health with customizable checks |
| **Resource limits** | Set CPU and memory constraints |
| **Rollback** | One command to revert to previous version (auto-timestamped tags) |
| **Dry run** | Preview changes before deploying |
| **Config file** | YAML config with environment variable expansion |
| **GitHub Action** | Built-in CI/CD integration |

## Documentation

| Guide | Description |
|-------|-------------|
| [Quick Start](docs/quickstart.md) | Deploy your first container in 5 minutes |
| [Configuration](docs/configuration.md) | All options and environment variables |
| [Examples](docs/examples.md) | Common deployment patterns |
| [GitHub Actions](docs/github-actions.md) | CI/CD integration |
| [Delta Transfer](docs/delta-transfer.md) | How layer caching works |
| [Flow Charts](docs/flow.md) | Visual deployment and rollback flows |

**Example Projects:** See [examples/](examples/) for complete, runnable examples.

## Requirements

- Docker installed locally and on remote server
- SSH access with key-based authentication

## Security

- SSH key-based authentication only
- Secrets passed via environment variables or env files
- No sensitive data logged
- Support for read-only containers and capability dropping

## Contributing

Contributions welcome! Please read our contributing guidelines.

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) for details.

---

<p align="center">
  <sub>Built for developers who just want to deploy</sub>
</p>
