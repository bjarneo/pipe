# Node.js Example

A simple Express.js web server demonstrating Pipe deployment with health checks.

## Files

- `server.js` - Express server with health endpoint
- `package.json` - Node.js dependencies
- `Dockerfile` - Multi-stage Docker build
- `pipe.yaml` - Pipe configuration

## Deploy

```bash
cd examples/node-app
pipe --dry-run  # Preview first
pipe            # Deploy
```

## Test

After deployment, test your server:

```bash
curl http://your-server.com:3000/
curl http://your-server.com:3000/health
```

## Configuration Highlights

This example demonstrates:
- Health check configuration
- Environment variables
- Build arguments with version tagging
- Resource limits
- Container labels
