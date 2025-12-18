# Example Projects

Ready-to-deploy example projects demonstrating Pipe in action.

## Available Examples

### [Node.js App](node-app/)

A production-ready Express.js server with:
- Health check endpoint (`/health`)
- Multi-stage Docker build
- Environment variables
- Resource limits
- Container labels

**Perfect for:** API servers, web applications, microservices

### [Static Site](static-site/)

A minimal nginx static site with:
- Read-only container (security hardened)
- tmpfs mounts for ephemeral data
- Simple health check

**Perfect for:** Landing pages, documentation sites, SPAs

## Quick Start

```bash
# 1. Choose an example
cd examples/node-app

# 2. Update the config with your server
vim pipe.yaml
# Change: host: your-server.com
#         user: your-username

# 3. Preview the deployment
pipe --dry-run

# 4. Deploy!
pipe
```

## Using Examples as Templates

Copy an example to start your own project:

```bash
cp -r examples/node-app my-project
cd my-project

# Customize for your app
vim Dockerfile
vim pipe.yaml

# Deploy
pipe
```

## Configuration Tips

### Dynamic Version Tags

Use shell variables for versioning:

```yaml
# pipe.yaml
tag: ${VERSION:-latest}
buildArgs:
  VERSION: ${VERSION:-dev}
  GIT_SHA: ${GIT_SHA}
```

```bash
VERSION=1.0.0 GIT_SHA=$(git rev-parse --short HEAD) pipe
```

### Multiple Environments

Create separate configs:

```
my-app/
├── pipe.yaml              # Development defaults
├── pipe.staging.yaml      # Staging overrides  
└── pipe.production.yaml   # Production settings
```

```bash
pipe --config pipe.production.yaml
```

### CI/CD Deployment

```bash
# In your CI pipeline
pipe --tag $CI_COMMIT_TAG --verbose
```

See [GitHub Actions docs](../docs/github-actions.md) for complete CI/CD examples.

## Need Help?

- [Quick Start Guide](../docs/quickstart.md)
- [Configuration Reference](../docs/configuration.md)
- [More Examples](../docs/examples.md)
