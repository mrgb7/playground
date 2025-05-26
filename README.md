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

Download the latest release for your platform. You can either download manually from the [releases page](https://github.com/mrgb7/playground/releases/latest) or use the following commands:

```bash
# Get the latest release information
LATEST_RELEASE=$(curl -s https://api.github.com/repos/mrgb7/playground/releases/latest | grep tag_name | cut -d '"' -f 4)

# Linux AMD64
curl -L -o playground.tar.gz https://github.com/mrgb7/playground/releases/download/${LATEST_RELEASE}/playground-${LATEST_RELEASE}-linux-amd64.tar.gz
tar -xzf playground.tar.gz
chmod +x playground-linux-amd64
sudo mv playground-linux-amd64 /usr/local/bin/playground

# macOS Intel
curl -L -o playground.tar.gz https://github.com/mrgb7/playground/releases/download/${LATEST_RELEASE}/playground-${LATEST_RELEASE}-darwin-amd64.tar.gz
tar -xzf playground.tar.gz
chmod +x playground-darwin-amd64
sudo mv playground-darwin-amd64 /usr/local/bin/playground

# macOS Apple Silicon
curl -L -o playground.tar.gz https://github.com/mrgb7/playground/releases/download/${LATEST_RELEASE}/playground-${LATEST_RELEASE}-darwin-arm64.tar.gz
tar -xzf playground.tar.gz
chmod +x playground-darwin-arm64
sudo mv playground-darwin-arm64 /usr/local/bin/playground
```

**Alternative: Quick install script**

```bash
# One-liner installation (detects your platform automatically)
curl -fsSL https://raw.githubusercontent.com/mrgb7/playground/main/install.sh | bash
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
# List all existing clusters
playground cluster list

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
playground cluster plugin add --name argocd --cluster my-cluster

# Install nginx ingress controller
playground cluster plugin add --name nginx-ingress --cluster my-cluster

# Install load balancer (MetalLB)
playground cluster plugin add --name load-balancer --cluster my-cluster

# Install ingress plugin (requires nginx and load-balancer)
# This plugin configures cluster domains and ArgoCD ingress
playground cluster plugin add --name ingress --cluster my-cluster

# Install TLS plugin (requires cert-manager)
# This plugin generates CA certificates and sets up cluster issuer
playground cluster plugin add --name tls --cluster my-cluster

# Uninstall a plugin
playground cluster plugin remove --name argocd --cluster my-cluster

# List available plugins
playground cluster plugin list

# Show plugin dependency information
playground cluster plugin deps --cluster my-cluster

# Show dependencies for a specific plugin
playground cluster plugin deps --cluster my-cluster --name ingress
```

#### Ingress Plugin

The ingress plugin provides domain-based access to your cluster services:

**Features:**
- Configures cluster domain: `{cluster-name}.local`
- Automatically sets up ArgoCD ingress if ArgoCD is installed
- Automatic TLS certificate generation when TLS plugin is installed
- Ensures nginx service is exposed as LoadBalancer
- Provides `/etc/hosts` configuration commands

**Dependencies:**
- `nginx-ingress` plugin must be installed
- `load-balancer` plugin must be installed

**Optional Enhancement:**
- `tls` plugin for automatic HTTPS certificate generation

**Usage:**
```bash
# Install dependencies first
playground cluster plugin add --name load-balancer --cluster my-cluster
playground cluster plugin add --name nginx-ingress --cluster my-cluster

# Install ingress plugin
playground cluster plugin add --name ingress --cluster my-cluster

# Optional: Install TLS plugin for HTTPS support
playground cluster plugin add --name cert-manager --cluster my-cluster
playground cluster plugin add --name tls --cluster my-cluster
# Now re-run ingress plugin to enable HTTPS
playground cluster plugin add --name ingress --cluster my-cluster
```

After installation, the plugin will provide commands to add entries to your `/etc/hosts` file for local domain access. If the TLS plugin is installed, ArgoCD will automatically be configured with HTTPS using the generated CA certificate.

#### TLS Plugin

The TLS plugin provides SSL/TLS certificate management for your cluster using self-signed CA certificates:

**Features:**
- Generates self-signed CA certificate for `*.{cluster-name}.local` domain
- Creates Kubernetes secret with CA certificate and private key
- Sets up cert-manager ClusterIssuer for automatic certificate generation
- Provides OS-specific instructions for trusting the CA certificate
- 10-year certificate validity period
- Supports macOS, Linux, and Windows trust store integration
- Automatic ArgoCD HTTPS configuration when used with ingress plugin

**Dependencies:**
- `cert-manager` plugin must be installed

**Usage:**
```bash
# Install cert-manager first
playground cluster plugin add --name cert-manager --cluster my-cluster

# Install TLS plugin
playground cluster plugin add --name tls --cluster my-cluster

# If ingress plugin is already installed, re-run it to enable HTTPS
playground cluster plugin add --name ingress --cluster my-cluster
```

**Integration with Ingress Plugin:**
When both TLS and ingress plugins are installed, the ingress plugin automatically:
- Detects the TLS cluster issuer (`local-ca-issuer`)
- Configures ArgoCD ingress with TLS certificate generation
- Updates ArgoCD to use HTTPS instead of HTTP
- Displays HTTPS URLs in the setup instructions

**Generated Resources:**
- Secret: `local-ca-secret` in `cert-manager` namespace
- ClusterIssuer: `local-ca-issuer`

**Using TLS Certificates:**
After installation, you can use the cluster issuer in your ingress resources:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-app
  annotations:
    cert-manager.io/cluster-issuer: local-ca-issuer
spec:
  tls:
  - hosts:
    - my-app.my-cluster.local
    secretName: my-app-tls
  rules:
  - host: my-app.my-cluster.local
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: my-app
            port:
              number: 80
```

The plugin will provide platform-specific commands to trust the CA certificate in your system's trust store.

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

For detailed system architecture, component interactions, and design diagrams, see [ARCHITECTURE.md](ARCHITECTURE.md).

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

## Quick Start

### Create a Cluster

```bash
# Create a simple single-node cluster
playground cluster create --name my-cluster

# Create a multi-node cluster
playground cluster create --name my-cluster --size 3

# Create a cluster with custom resource specifications
playground cluster create --name my-cluster --size 3 \
  --master-cpus 4 --master-memory 4G --master-disk 50G \
  --worker-cpus 2 --worker-memory 4G --worker-disk 30G

# Create a cluster with core components
playground cluster create --name my-cluster --size 3 --with-core-component
```

### Cluster Resource Configuration

You can customize the CPU, memory, and disk resources for master and worker nodes:

**Master Node Configuration:**
- `--master-cpus`: Number of CPUs (1-32, default: 2)
- `--master-memory`: Memory allocation (format: `2G`, `1024M`, default: `2G`)
- `--master-disk`: Disk size (format: `20G`, `1024M`, `1T`, default: `20G`)

**Worker Node Configuration:**
- `--worker-cpus`: Number of CPUs per worker (1-32, default: 2)
- `--worker-memory`: Memory per worker (format: `2G`, `1024M`, default: `2G`)
- `--worker-disk`: Disk size per worker (format: `20G`, `1024M`, `1T`, default: `20G`)

**Examples:**
```bash
# High-performance cluster
playground cluster create --name perf-cluster --size 5 \
  --master-cpus 8 --master-memory 8G --master-disk 100G \
  --worker-cpus 4 --worker-memory 8G --worker-disk 50G

# Development cluster with minimal resources
playground cluster create --name dev-cluster --size 2 \
  --master-cpus 1 --master-memory 1G --master-disk 10G \
  --worker-cpus 1 --worker-memory 1G --worker-disk 10G

# Mixed workload cluster
playground cluster create --name mixed-cluster --size 3 \
  --master-cpus 4 --master-memory 4G --master-disk 30G \
  --worker-cpus 2 --worker-memory 2G --worker-disk 20G
```

### Default Cluster Specifications

When no custom resources are specified, the following defaults are used:

- **Master Node**: 2 CPUs, 2GB RAM, 20GB disk
- **Worker Nodes**: 2 CPUs, 2GB RAM, 20GB disk
- **K3s Version**: Latest stable
- **Disabled Components**: servicelb, traefik (to avoid conflicts)

**Resource Limits:**
- CPU: 1-32 cores per node
- Memory: Must be specified with `G` (GB) or `M` (MB) suffix
- Disk: Must be specified with `G` (GB), `M` (MB), or `T` (TB) suffix

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

This project is licensed under the MIT License.

## Support

For issues and questions:
- Open an issue on GitHub
- Check the troubleshooting section
- Review existing issues and discussions