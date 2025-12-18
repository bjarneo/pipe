# Use the official nginx image as base
FROM nginx:alpine

# Create index.html directly in the container
RUN echo '<!DOCTYPE html>\
<html>\
<head>\
    <title>Welcome to pipe3</title>\
</head>\
<body>\
    <div class="container">\
        <h1>Welcome to pipe!</h1>\
        <p>If you see this page, the nginx web server is successfully installed and working.</p>\
        <p>This page was created from within the Dockerfile.</p>\
        <p>This is something else.</p>\
    </div>\
</body>\
</html>' > /usr/share/nginx/html/index.html

# Configure nginx to listen on port 5473
RUN sed -i 's/listen\s*80;/listen 5473;/g' /etc/nginx/conf.d/default.conf

# Expose port 5473
EXPOSE 5473

# Start nginx in the foreground
CMD ["nginx", "-g", "daemon off;"]
