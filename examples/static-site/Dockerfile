FROM nginx:alpine

# Copy static files
COPY index.html /usr/share/nginx/html/

# Nginx runs on port 80
EXPOSE 80

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost/ || exit 1
