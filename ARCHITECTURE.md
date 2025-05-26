# Playground Architecture

A CLI tool for creating K3s clusters on Multipass VMs with a plugin system.

## How It Works

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│     CLI     │───▶│  Business   │───▶│Infrastructure│
│ (Commands)  │    │   Logic     │    │(VMs + K3s)  │
└─────────────┘    └─────────────┘    └─────────────┘
```

## Project Structure

```
playground/
├── cmd/                    # CLI commands
│   ├── root/              # Main command
│   └── cluster/           # Cluster management
│       └── plugin/        # Plugin commands
├── internal/              # Core logic
│   ├── multipass/         # VM management
│   ├── k8s/              # Kubernetes operations
│   ├── plugins/          # Plugin system
│   └── installer/        # K3s installation
└── pkg/
    └── logger/           # Logging utilities
```

## Cluster Creation Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│ playground cluster create --name my-cluster --size 3                   │
└─────────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   CLI Parser    │───▶│ Cluster Manager │───▶│ Multipass API   │
│ cmd/cluster/    │    │ internal/       │    │ External Tool   │
│ create.go       │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                  │                       │
                                  ▼                       ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │ K3s Installer   │    │  Ubuntu VMs     │
                       │ internal/       │    │ my-cluster-*    │
                       │ installer/      │    │ (3 VMs created) │
                       └─────────────────┘    └─────────────────┘
                                  │                       │
                                  ▼                       ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │ Kubeconfig      │    │ K3s Cluster     │
                       │ Setup           │    │ master + 2      │
                       │ ~/.kube/config  │    │ workers running │
                       └─────────────────┘    └─────────────────┘
```

## Plugin Add/Remove Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│ playground cluster plugin add --name argocd --cluster my-cluster       │
└─────────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ Plugin Command  │───▶│ Plugin Manager  │───▶│ Plugin Factory  │
│ cmd/cluster/    │    │ internal/       │    │ internal/       │
│ plugin/add.go   │    │ plugins/        │    │ plugins/        │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                  │                       │
                                  ▼                       ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │ Dependency      │    │ ArgoCD Plugin   │
                       │ Resolver        │    │ argocd.go       │
                       │ Check: none     │    │ Implements:     │
                       │ for ArgoCD      │    │ Plugin interface│
                       └─────────────────┘    └─────────────────┘
                                  │                       │
                                  ▼                       ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │ Helm Client     │    │ K8s Client      │
                       │ internal/k8s/   │    │ internal/k8s/   │
                       │ Install chart   │    │ Apply resources │
                       └─────────────────┘    └─────────────────┘
                                  │                       │
                                  ▼                       ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │ Installation    │    │ K3s Cluster     │
                       │ Tracker         │    │ ArgoCD pods     │
                       │ Save state      │    │ running in      │
                       │ in memory       │    │ argocd namespace│
                       └─────────────────┘    └─────────────────┘

For REMOVE: Same flow but calls Uninstall() → helm uninstall → delete resources
```

## Plugin Dependencies

```
Independent:          Dependent:
┌─────────────┐      ┌─────────────┐
│   ArgoCD    │      │   Nginx     │ (needs MetalLB)
│ MetalLB     │      │   Ingress   │ (needs Nginx+MetalLB)
│ CertManager │      │    TLS      │ (needs CertManager)
└─────────────┘      └─────────────┘
```

## Available Plugins

- **ArgoCD**: GitOps deployment
- **Nginx**: Ingress controller
- **MetalLB**: Load balancer for VMs
- **Ingress**: Domain routing (needs Nginx+MetalLB)
- **TLS**: SSL certificates (needs CertManager)
- **CertManager**: Certificate management

## Tech Stack

- **Go**: Main language
- **Multipass**: VM management
- **K3s**: Lightweight Kubernetes
- **Helm**: Package management
- **Cobra**: CLI framework

## Key Files

- `main.go`: Entry point
- `cmd/cluster/create.go`: Cluster creation logic
- `cmd/cluster/plugin/`: Plugin commands
- `internal/plugins/`: Plugin implementations
- `internal/multipass/`: VM operations
- `internal/k8s/`: Kubernetes client operations

## Common Workflows

**Full setup:**
```bash
# Create cluster with core components
playground cluster create --name dev --with-core-components

# Add ArgoCD
playground cluster plugin add --name argocd --cluster dev

# Add ingress (installs nginx+metallb automatically)
playground cluster plugin add --name ingress --cluster dev
```

**Cleanup:**
```bash
playground cluster delete --name dev
playground cluster clean  # Remove all clusters
```

That's it. The tool creates VMs, installs K3s, and manages plugins through Helm charts. 