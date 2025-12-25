#!/bin/bash

# Function to get latest release version
get_latest_version() {
    curl -sS https://api.github.com/repos/bjarneo/pipe/releases/latest | grep "tag_name" | cut -d '"' -f 4
}

# Function to detect OS and architecture
detect_system() {
    local os
    local arch

    # Detect OS
    case "$(uname -s)" in
        Darwin*)  os="darwin" ;;
        Linux*)   os="linux" ;;
        *)        echo "Unsupported operating system" && exit 1 ;;
    esac

    # Detect architecture
    case "$(uname -m)" in
        x86_64)  arch="amd64" ;;
        arm64)   arch="arm64" ;;
        aarch64) arch="arm64" ;;
        *)       echo "Unsupported architecture" && exit 1 ;;
    esac

    echo "${os}-${arch}"
}

# Main installation process
main() {
    echo "Detecting system..."
    local system=$(detect_system)
    local version=$(get_latest_version)
    
    echo "Latest version: ${version}"
    echo "System detected: ${system}"
    
    # Check if pipe already exists
    if command -v pipe >/dev/null 2>&1; then
        echo "pipe is already installed at $(which pipe)"
        # Only prompt if running interactively (not piped)
        if [ -t 0 ]; then
            read -p "Do you want to override the existing installation? (y/N) " response
            case "$response" in
                [yY][eE][sS]|[yY])
                    echo "Proceeding with installation..."
                    ;;
                *)
                    echo "Installation cancelled"
                    exit 0
                    ;;
            esac
        else
            echo "Upgrading..."
        fi
    fi
    
    local binary_name="pipe-${system}"
    local download_url="https://github.com/bjarneo/pipe/releases/download/${version}/${binary_name}"
    
    echo "Downloading pipe..."
    if ! curl -sSL -o pipe "${download_url}"; then
        echo "Failed to download pipe"
        exit 1
    fi
    
    echo "Making binary executable..."
    chmod +x pipe
    
    echo "Moving to /usr/local/bin..."
    if ! sudo mv pipe /usr/local/bin/pipe; then
        echo "Failed to move pipe to /usr/local/bin"
        echo "Please run with sudo or check permissions"
        exit 1
    fi
    
    echo "pipe has been successfully installed!"
    echo "You can now use it by running: pipe --help"
}

main
