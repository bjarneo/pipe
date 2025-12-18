# Delta Layer Transfer

Pipe uses **delta transfers** to dramatically speed up deployments after the first one.

## The Problem

Docker images can be large-hundreds of megabytes or even gigabytes. Transferring the entire image on every deployment wastes time and bandwidth.

## The Solution

Docker images are built in layers. When you update your code, typically only the top few layers change-your base image layers (Node.js, Python, nginx, etc.) stay the same.

Pipe takes advantage of this:

```
+-------------------------------------+
|  Your code (CHANGED)                |  <-- Only this transfers
+-------------------------------------+
|  npm install / dependencies         |  <-- Cached remotely
+-------------------------------------+
|  Node.js runtime                    |  <-- Cached remotely  
+-------------------------------------+
|  Alpine Linux base                  |  <-- Cached remotely
+-------------------------------------+
```

## How It Works

On each deployment, Pipe:

1. **Inspects** local image layers
2. **Queries** the remote host for cached layers
3. **Transfers** only new/changed layers
4. **Reconstructs** the full image on the remote

## Real-World Impact

| Scenario | Full Transfer | Delta Transfer | Savings |
|----------|---------------|----------------|---------|
| Code change only | 500 MB | 5 MB | 99% |
| Dependency update | 500 MB | 50 MB | 90% |
| Base image update | 500 MB | 200 MB | 60% |
| First deployment | 500 MB | 500 MB | 0% |

## What You'll See

```bash
$ pipe

> Deploying my-app:latest to server.com
> Building Docker image
> Transferring image to remote host
  Layer cache: 8/10 layers cached (80%), transferring 2 changed layers
> Starting container
* Deployment completed successfully!
```

## Tips for Maximum Cache Hits

### 1. Order Dockerfile Instructions Wisely

Put things that change frequently at the **bottom**:

```dockerfile
# Good - dependencies cached separately
FROM node:20-alpine
WORKDIR /app
COPY package*.json ./          # Changes rarely
RUN npm ci                     # Changes rarely
COPY . .                       # Changes often
CMD ["node", "server.js"]
```

```dockerfile
# ❌ Bad - cache invalidated on every code change
FROM node:20-alpine
WORKDIR /app
COPY . .                       # This invalidates everything below
RUN npm ci                     # Reinstalls every time
CMD ["node", "server.js"]
```

### 2. Use .dockerignore

Exclude files that shouldn't trigger rebuilds:

```
# .dockerignore
node_modules
.git
*.md
.env*
```

### 3. Use Multi-Stage Builds

Keep your final image small:

```dockerfile
# Build stage
FROM node:20 AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

# Production stage (smaller, fewer layers)
FROM node:20-alpine
WORKDIR /app
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/node_modules ./node_modules
CMD ["node", "dist/server.js"]
```

## Fallback Behavior

If delta transfer fails for any reason, Pipe automatically falls back to a full transfer. You'll never have a failed deployment due to caching issues.

## Verbose Output

Want to see exactly what's happening?

```bash
pipe --verbose
```

This shows:
- Which layers are being compared
- Cache hit/miss for each layer
- Actual bytes transferred
