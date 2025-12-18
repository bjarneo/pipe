const express = require('express');
const app = express();

const PORT = process.env.PORT || 3000;
const VERSION = process.env.VERSION || 'unknown';
const NODE_ENV = process.env.NODE_ENV || 'development';

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({
    status: 'healthy',
    version: VERSION,
    environment: NODE_ENV,
    uptime: process.uptime(),
    timestamp: new Date().toISOString()
  });
});

// Main endpoint
app.get('/', (req, res) => {
  res.json({
    message: 'Hello from Pipe!',
    version: VERSION,
    environment: NODE_ENV
  });
});

// Info endpoint
app.get('/info', (req, res) => {
  res.json({
    name: 'node-app',
    version: VERSION,
    node: process.version,
    platform: process.platform,
    memory: process.memoryUsage(),
    uptime: process.uptime()
  });
});

app.listen(PORT, '0.0.0.0', () => {
  console.log(`Server running on port ${PORT}`);
  console.log(`Environment: ${NODE_ENV}`);
  console.log(`Version: ${VERSION}`);
});
