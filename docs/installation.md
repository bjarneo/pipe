# Installation

Get Pipe up and running in minutes.

## One-Line Install

The fastest way to install Pipe:

```bash
curl -fsSL https://raw.githubusercontent.com/bjarneo/pipe/main/install.sh | sh
```

This automatically detects your OS and architecture, downloads the correct binary, and installs it to `/usr/local/bin`.

## Download Binary

Prefer to install manually? Grab a pre-built binary from the [releases page](https://github.com/bjarneo/pipe/releases).

| Platform | Architecture | Download |
|----------|--------------|----------|
| Linux    | x86_64       | `pipe-linux-amd64` |
| Linux    | ARM64        | `pipe-linux-arm64` |
| macOS    | Intel        | `pipe-darwin-amd64` |
| macOS    | Apple Silicon| `pipe-darwin-arm64` |

```bash
# Download (replace with your platform)
curl -LO https://github.com/bjarneo/pipe/releases/latest/download/pipe-linux-amd64

# Make executable
chmod +x pipe-linux-amd64

# Move to PATH
sudo mv pipe-linux-amd64 /usr/local/bin/pipe

# Verify installation
pipe --version
```

## Build from Source

For the latest development version or if you want to contribute:

**Requirements:** Go 1.21+

```bash
git clone https://github.com/bjarneo/pipe.git
cd pipe
go build -o pipe .
sudo mv pipe /usr/local/bin/
```

## Verify Installation

```bash
pipe --version
pipe --help
```

## What's Next?

- [Quick Start Guide](quickstart.md) - Deploy your first container in 5 minutes
- [Configuration](configuration.md) - All available options
- [Examples](../examples/) - Real-world deployment examples

## Troubleshooting

### "command not found"

Make sure `/usr/local/bin` is in your PATH:

```bash
echo $PATH | grep -q '/usr/local/bin' && echo "OK" || echo "Add /usr/local/bin to PATH"
```

### Permission denied

If you can't write to `/usr/local/bin`, install to a user directory:

```bash
mkdir -p ~/.local/bin
mv pipe ~/.local/bin/
export PATH="$HOME/.local/bin:$PATH"
```

Add the export line to your `~/.bashrc` or `~/.zshrc` to make it permanent.
