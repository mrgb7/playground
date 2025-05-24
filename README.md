# Playground - K3s Cluster Management Tool

A CLI tool for creating and managing K3s clusters using Multipass VMs. This tool simplifies the process of setting up local Kubernetes clusters for development and testing purposes.

## Features

- **Cluster Management**: Create, delete, and manage K3s clusters
- **Multipass Integration**: Uses Multipass VMs as cluster nodes
- **Plugin Support**: Install and manage various Kubernetes plugins
- **ArgoCD Integration**: Built-in support for ArgoCD deployment
- **Kubeconfig Management**: Automatic kubeconfig setup and merging

## Prerequisites

- [Multipass](https://multipass.run/) installed and available in PATH
- Go 1.21+ for building from source
- Sufficient system resources for running VMs

## Installation

### From Source

```bash
git clone https://github.com/mrgb7/playground.git
cd playground
go build -o playground .
```

### Using Go Install

```bash
go install github.com/mrgb7/playground@latest
```

## Usage

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

## Development

### Building

```bash
go build -o playground .
```

### Running Tests

```bash
go test ./...
```

### Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

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

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For issues and questions:
- Open an issue on GitHub
- Check the troubleshooting section
- Review existing issues and discussions