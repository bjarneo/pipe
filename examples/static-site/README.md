# Static Site Example

A minimal nginx static site demonstrating basic Pipe deployment.

## Files

- `index.html` - Simple HTML page
- `Dockerfile` - Nginx-based container
- `pipe.yaml` - Pipe configuration

## Deploy

```bash
cd examples/static-site
pipe --dry-run  # Preview first
pipe            # Deploy
```

## Test

```bash
curl http://your-server.com:8080/
```

## Configuration Highlights

This example demonstrates:
- Minimal configuration
- Read-only container
- Nginx health check
