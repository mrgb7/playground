# Playground - K3s Cluster Management Tool

[![GitHub Release](https://img.shields.io/github/release/mrgb7/playground.svg)](https://github.com/mrgb7/playground/releases)
[![CI](https://github.com/mrgb7/playground/workflows/Pull%20Request/badge.svg)](https://github.com/mrgb7/playground/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/mrgb7/playground)](https://goreportcard.com/report/github.com/mrgb7/playground)

A CLI tool for creating and managing K3s clusters using Multipass VMs. This tool simplifies the process of setting up local Kubernetes clusters for development and testing purposes.

## Features

- **Cluster Management**: Create, delete, and manage K3s clusters
- **Multipass Integration**: Uses Multipass VMs as cluster nodes
- **Plugin Support**: Install and manage various Kubernetes plugins
- **ArgoCD Integration**: Built-in support for ArgoCD deployment
- **Kubeconfig Management**: Automatic kubeconfig setup and merging

## Prerequisites

- [Multipass](https://multipass.run/) installed and available in PATH
- Go 1.24+ for building from source
- Sufficient system resources for running VMs

## Installation

### Pre-built Binaries

Download the latest release for your platform:

```bash
# Linux AMD64
curl -L -o playground.tar.gz https://github.com/mrgb7/playground/releases/latest/download/playground-latest-linux-amd64.tar.gz
tar -xzf playground.tar.gz
chmod +x playground-linux-amd64
sudo mv playground-linux-amd64 /usr/local/bin/playground

# macOS Intel
curl -L -o playground.tar.gz https://github.com/mrgb7/playground/releases/latest/download/playground-latest-darwin-amd64.tar.gz
tar -xzf playground.tar.gz
chmod +x playground-darwin-amd64
sudo mv playground-darwin-amd64 /usr/local/bin/playground

# macOS Apple Silicon
curl -L -o playground.tar.gz https://github.com/mrgb7/playground/releases/latest/download/playground-latest-darwin-arm64.tar.gz
tar -xzf playground.tar.gz
chmod +x playground-darwin-arm64
sudo mv playground-darwin-arm64 /usr/local/bin/playground
```

### From Source

```bash
git clone https://github.com/mrgb7/playground.git
cd playground
make build
# Binary will be in bin/playground
```

### Using Go Install

```bash
go install github.com/mrgb7/playground@latest
```

## Usage

### Version Information

```bash
# Check version
playground version

# Detailed version info
playground version --verbose
```

### Basic Commands

```bash
# Create a single-node cluster
playground cluster create --name my-cluster

# Create a multi-node cluster
playground cluster create --name my-cluster --size 3

# Create cluster with core components
playground cluster create --name my-cluster --with-core-component

# Delete a cluster
playground cluster delete --name my-cluster

# Clean up all resources
playground cluster clean
```

### Plugin Management

```bash
# Install ArgoCD plugin
playground cluster plugin install argocd --cluster my-cluster

# Uninstall a plugin
playground cluster plugin uninstall argocd --cluster my-cluster

# List available plugins
playground cluster plugin list
```

## Development

### Setup

```bash
git clone https://github.com/mrgb7/playground.git
cd playground
make dev-setup
```

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Build release binaries (Linux & macOS)
make build-release
```

### Testing

```bash
# Run tests
make test

# Run tests with coverage
make test-coverage

# Run tests with race detection
make test-race
```

### Code Quality

```bash
# Format code
make fmt

# Lint code
make lint

# Run all pre-commit checks
make pre-commit
```

## CI/CD Pipeline

This project uses GitHub Actions for continuous integration and delivery:

### Pull Request Workflow
- **Code Formatting**: Ensures `gofmt` compliance
- **Linting**: Runs `golangci-lint` for code quality
- **Testing**: Unit tests with race detection and coverage
- **Security Scanning**: Vulnerability analysis with Gosec
- **Multi-platform Build**: Validates builds on Linux, macOS, and Windows

### Release Workflow
- **Semantic Versioning**: Automatic version bumping based on conventional commits
- **Release Builds**: Binaries for Linux AMD64, macOS Intel, and macOS Apple Silicon
- **GitHub Releases**: Automated release creation with changelogs
- **Asset Publishing**: Packaged binaries as downloadable assets

### Conventional Commits

We use [Conventional Commits](https://www.conventionalcommits.org/) for automatic versioning:

- `feat:` - New feature (minor version bump)
- `fix:` - Bug fix (patch version bump)
- `feat!:` or `BREAKING CHANGE:` - Breaking change (major version bump)

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

## Architecture

The tool is organized into several packages:

- `cmd/`: CLI commands and subcommands
- `internal/`: Internal packages not meant for external use
  - `multipass/`: Multipass client for VM management
  - `k8s/`: Kubernetes client utilities
  - `installer/`: Package installers (Helm, etc.)
  - `plugins/`: Plugin implementations
- `pkg/`: Public packages that can be imported
  - `logger/`: Colored logging utilities
- `types/`: Shared type definitions

## Configuration

The tool uses sensible defaults but can be configured through:

- Command-line flags
- Environment variables
- Configuration files (planned)

### Default Cluster Specifications

- **Master Node**: 2 CPUs, 2GB RAM, 10GB disk
- **Worker Nodes**: 1 CPU, 1GB RAM, 5GB disk
- **K3s Version**: Latest stable
- **Disabled Components**: servicelb, traefik (to avoid conflicts)

## Security Considerations

- Kubeconfig files are temporarily stored in system temp directory
- K3s installation uses official installation scripts
- Plugin values are fetched from trusted sources

## Troubleshooting

### Common Issues

1. **Multipass not found**: Ensure Multipass is installed and in PATH
2. **VM creation fails**: Check available system resources
3. **K3s installation timeout**: Increase timeout or check network connectivity
4. **Kubeconfig merge fails**: Check file permissions and existing config

### Debug Mode

Enable debug logging by setting:
```bash
export LOG_LEVEL=debug
```

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on:

- Development setup
- Commit message format
- Pull request process
- Release workflow

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For issues and questions:
- Open an issue on GitHub
- Check the troubleshooting section
- Review existing issues and discussions