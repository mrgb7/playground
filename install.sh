#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Detect platform
detect_platform() {
    local os
    local arch
    
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    arch=$(uname -m)
    
    case "$os" in
        linux)
            os="linux"
            ;;
        darwin)
            os="darwin"
            ;;
        *)
            print_error "Unsupported operating system: $os"
            exit 1
            ;;
    esac
    
    case "$arch" in
        x86_64|amd64)
            arch="amd64"
            ;;
        arm64|aarch64)
            arch="arm64"
            ;;
        *)
            print_error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac
    
    echo "${os}-${arch}"
}

# Get latest release tag
get_latest_release() {
    curl -s https://api.github.com/repos/mrgb7/playground/releases/latest | grep tag_name | cut -d '"' -f 4
}

# Download and install binary
install_playground() {
    local platform="$1"
    local version="$2"
    local temp_dir
    
    temp_dir=$(mktemp -d)
    cd "$temp_dir"
    
    local binary_name="playground-${platform}"
    local archive_name="playground-${version}-${platform}.tar.gz"
    local download_url="https://github.com/mrgb7/playground/releases/download/${version}/${archive_name}"
    
    print_status "Downloading playground ${version} for ${platform}..."
    if ! curl -L -o "$archive_name" "$download_url"; then
        print_error "Failed to download $download_url"
        exit 1
    fi
    
    print_status "Extracting archive..."
    if ! tar -xzf "$archive_name"; then
        print_error "Failed to extract archive"
        exit 1
    fi
    
    if [[ ! -f "$binary_name" ]]; then
        print_error "Binary $binary_name not found in archive"
        exit 1
    fi
    
    chmod +x "$binary_name"
    
    # Determine install location
    local install_dir="/usr/local/bin"
    if [[ ! -w "$install_dir" ]]; then
        print_status "Installing to $install_dir (requires sudo)..."
        sudo mv "$binary_name" "$install_dir/playground"
    else
        print_status "Installing to $install_dir..."
        mv "$binary_name" "$install_dir/playground"
    fi
    
    # Clean up
    cd /
    rm -rf "$temp_dir"
    
    print_success "playground installed successfully!"
    print_status "Run 'playground version' to verify the installation"
}

# Check dependencies
check_dependencies() {
    if ! command -v curl >/dev/null 2>&1; then
        print_error "curl is required but not installed"
        exit 1
    fi
    
    if ! command -v tar >/dev/null 2>&1; then
        print_error "tar is required but not installed"
        exit 1
    fi
}

# Main installation function
main() {
    print_status "Starting playground installation..."
    
    check_dependencies
    
    local platform
    platform=$(detect_platform)
    print_status "Detected platform: $platform"
    
    local version
    version=$(get_latest_release)
    print_status "Latest release: $version"
    
    install_playground "$platform" "$version"
    
    # Verify installation
    if command -v playground >/dev/null 2>&1; then
        print_success "Installation verified!"
        playground version
    else
        print_warning "Installation completed but 'playground' command not found in PATH"
        print_status "You may need to restart your shell or add /usr/local/bin to your PATH"
    fi
}

# Run main function
main "$@" 