# Playground Cluster Architecture

**Playground** is a CLI tool for creating and managing lightweight K3s Kubernetes clusters using Multipass VMs with an extensible plugin system. This document outlines the architectural design and component interactions.

## Cluster Bootstrapping Flow

```
┌─────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                         Cluster Creation Flow                                           │
└─────────────────────────────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────┐  1. Cluster Create    ┌─────────────────┐  2. VM Launch      ┌─────────────────┐
│   Playground    │ ────────────────────► │   Multipass     │ ─────────────────► │ Virtualization  │
│      CLI        │                       │    Engine       │                    │     Layer       │
│                 │                       │                 │                    │                 │
│ • Parse Args    │                       │ • VM Lifecycle  │                    │ • Hypervisor    │
│ • Validate      │                       │ • Network Mgmt  │                    │ • Host Resources│
│ • Orchestrate   │                       │ • SSH/Exec      │                    │ • Platform APIs │
└─────────────────┘                       └─────────────────┘                    └─────────────────┘
                                                   │                                       │
                                                   │ 3. VM Creation                        │
                                                   ▼                                       ▼
                                          ┌─────────────────────────────────────────────────────────┐
                                          │                 Host System                              │
                                          │                                                         │
                                          │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐   │
                                          │  │   Master    │  │  Worker-1   │  │  Worker-2   │   │
                                          │  │     VM      │  │     VM      │  │     VM      │   │
                                          │  │             │  │             │  │             │   │
                                          │  │Ubuntu 22.04 │  │Ubuntu 22.04 │  │Ubuntu 22.04 │   │
                                          │  │2CPU/4GB RAM │  │2CPU/4GB RAM │  │2CPU/4GB RAM │   │
                                          │  │20GB Storage │  │20GB Storage │  │20GB Storage │   │
                                          │  └─────────────┘  └─────────────┘  └─────────────┘   │
                                          └─────────────────────────────────────────────────────────┘
                                                                        │
                                                                        │ 4. K3s Installation
                                                                        ▼
┌──────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                    K3s Bootstrap Process                                                 │
│                                                                                                          │
│  ┌─────────────────┐  Install K3s   ┌─────────────────┐  Join Token    ┌─────────────────┐              │
│  │   Playground    │ ─────────────► │   Master VM     │ ──────────────► │   Worker VMs    │              │
│  │                 │                │                 │                 │                 │              │
│  │ • SSH to Master │                │ • K3s Server    │                 │ • K3s Agent     │              │
│  │ • Install K3s   │                │ • API Server    │                 │ • kubelet       │              │
│  │ • Get Token     │                │ • etcd          │                 │ • containerd    │              │
│  │ • SSH to Workers│                │ • Scheduler     │                 │ • flannel CNI   │              │
│  │ • Join Cluster  │                │ • Controller    │                 │                 │              │
│  └─────────────────┘                │ • containerd    │                 │                 │              │
│                                     │ • flannel CNI   │                 │                 │              │
│                                     └─────────────────┘                 └─────────────────┘              │
└──────────────────────────────────────────────────────────────────────────────────────────────────────────┘
                                                  │
                                                  │ 5. Kubeconfig Setup
                                                  ▼
┌──────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                    Final Architecture State                                              │
│                                                                                                          │
│  ┌─────────────────┐  HTTPS/TLS     ┌───────────────────────────────────────────────────────────────┐   │
│  │   Developer     │ ──────────────► │                    K3s Cluster                               │   │
│  │  Workstation    │  Port 6443     │                                                               │   │
│  │                 │                │  ┌─────────────┐         ┌─────────────┐  ┌─────────────┐    │   │
│  │ • kubectl       │                │  │ Master Node │         │ Worker-1    │  │ Worker-2    │    │   │
│  │ • playground    │                │  │             │         │             │  │             │    │   │
│  │ • ~/.kube/config│                │  │ • API Server│         │ • kubelet   │  │ • kubelet   │    │   │
│  │                 │                │  │ • etcd      │         │ • pods      │  │ • pods      │    │   │
│  └─────────────────┘                │  │ • pods      │         │ • services  │  │ • services  │    │   │
│                                     │  │ • services  │         │             │  │             │    │   │
│                                     │  └─────────────┘         └─────────────┘  └─────────────┘    │   │
│                                     └───────────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

## Plugin Dependency Graph Architecture

### Unified Dependency Graph Structure

```
┌──────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                 DependencyGraph Core Structure                                           │
│                                                                                                          │
│  ┌─────────────────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                                    GraphNode                                                       │ │
│  │                                                                                                     │ │
│  │   ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐                                │ │
│  │   │     Plugin      │    │  Dependencies   │    │   Dependents    │                                │ │
│  │   │   Interface     │    │    []string     │    │    []string     │                                │ │
│  │   │                 │    │                 │    │                 │                                │ │
│  │   │ • Name()        │    │ • What this     │    │ • What plugins  │                                │ │
│  │   │ • Install()     │    │   plugin needs  │    │   depend on     │                                │ │
│  │   │ • Uninstall()   │    │ • Resolved at   │    │   this plugin   │                                │ │
│  │   │ • Status()      │    │   AddPlugin()   │    │ • Updated when  │                                │ │
│  │   │ • GetDepends()  │    │                 │    │   dependencies  │                                │ │
│  │   └─────────────────┘    └─────────────────┘    │   are added     │                                │ │
│  │                                                 └─────────────────┘                                │ │
│  └─────────────────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                                          │
│  Graph Operations:                                                                                       │
│  • AddPlugin(plugin) → Build bidirectional relationships                                                │
│  • GetInstallOrder(plugins) → Topological sort for dependency resolution                               │
│  • GetUninstallOrder(plugins) → Reverse topological sort for safe removal                              │
│  • ValidateInstall/Uninstall(plugin) → Check dependency constraints                                    │
│  • HasCycles() → Detect circular dependencies using DFS                                                │
└──────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

### Plugin Dependency Relationships

```
┌──────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                     Plugin Dependency Map                                               │
│                                                                                                          │
│                                ┌─────────────────┐                                                      │
│                                │   argocd (A)    │ ◄── Independent                                      │
│                                │ Dependencies:[] │                                                      │
│                                │ Dependents: []  │                                                      │
│                                └─────────────────┘                                                      │
│                                                                                                          │
│                                ┌─────────────────┐                                                      │
│                                │cert-manager (CM)│ ◄── Independent                                      │
│                                │ Dependencies:[] │                                                      │
│          ┌─────────────────────┤ Dependents:[T]  │                                                      │
│          │                     └─────────────────┘                                                      │
│          │                                                                                              │
│          │                     ┌─────────────────┐                                                      │
│          └────────────────────►│    tls (T)      │ ◄── Depends on cert-manager                         │
│                                │ Dependencies:[CM│                                                      │
│                                │ Dependents: []  │                                                      │
│                                └─────────────────┘                                                      │
│                                                                                                          │
│  ┌─────────────────┐           ┌─────────────────┐           ┌─────────────────┐                        │
│  │load-balancer(LB)│ ◄─────────┤   nginx (N)     │ ◄─────────┤   ingress (I)   │ ◄── Complex Chain      │
│  │ Dependencies:[] │           │ Dependencies:[LB│           │ Dependencies:[N,│                        │
│  │ Dependents:[N,I]│           │ Dependents: [I] │           │                LB]                       │
│  └─────────────────┘           └─────────────────┘           │ Dependents: []  │                        │
│                                                               └─────────────────┘                        │
└──────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

### Dependency Resolution Flow

```
┌──────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                Plugin Installation with Dependency Resolution                           │
└──────────────────────────────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────┐  1. User Request   ┌─────────────────────────────────────────────────────────────────┐
│   playground    │ ─────────────────► │                   DependencyValidator                           │
│  plugin add     │   install "ingress"│                                                                 │
│   --name        │                    │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│   ingress       │                    │  │   Validation    │  │  Dependency     │  │ Order           │ │
│                 │                    │  │   Pipeline      │  │  Resolution     │  │ Calculation     │ │
└─────────────────┘                    │  │                 │  │                 │  │                 │ │
                                       │  │ • Check plugin  │  │ • Build dep     │  │ • Topological   │ │
                                       │  │   exists        │  │   graph         │  │   sort          │ │
                                       │  │ • Validate      │  │ • Find all      │  │ • Cycle         │ │
                                       │  │   constraints   │  │   dependencies  │  │   detection     │ │
                                       │  │ • Check cycles  │  │ • Resolve order │  │ • Filter        │ │
                                       │  └─────────────────┘  └─────────────────┘  │   installed     │ │
                                       │                                           └─────────────────┘ │
                                       └──────────────────────┬──────────────────────────────────────────┘
                                                             │
                                                             ▼
┌──────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                    Resolution Result                                                     │
│                                                                                                          │
│  Target: "ingress"                                                                                       │
│  Dependencies Found: ["nginx", "load-balancer"]                                                         │
│  Install Order: ["load-balancer", "nginx", "ingress"]                                                   │
│  Already Installed: []                                                                                  │
│  Final Install Order: ["load-balancer", "nginx", "ingress"]                                             │
└──────────────────────────────────────────────────────────────────────────────────────────────────────────┘
                                                             │
                                                             ▼
┌──────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                   Plugin Installation                                                   │
│                                                                                                          │
│  ┌─────────────────┐  Install      ┌─────────────────┐  Install      ┌─────────────────┐               │
│  │ load-balancer   │ ─────────────► │     nginx       │ ─────────────► │    ingress      │               │
│  │                 │   Step 1      │                 │   Step 2      │                 │   Step 3      │
│  │ • MetalLB       │               │ • Nginx Ingress │               │ • Ingress       │               │
│  │ • LoadBalancer  │               │ • Controller    │               │ • Controller    │               │
│  │   Service       │               │ • ConfigMap     │               │ • Rules         │               │
│  └─────────────────┘               └─────────────────┘               └─────────────────┘               │
└──────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

## Plugin Management Flow

### Plugin Installation Process

```
                                    ┌─────────────────┐
                                    │   Add Plugin    │
                                    └─────────┬───────┘
                                              │
                                              ▼
                                    ┌─────────────────┐
                                    │ DependencyValidator│
                                    │ ValidateInstall() │
                                    └─────┬─────┬─────┘
                                          │     │
                              ┌───────────▼     ▼─────────────┐
                              │                               │
                    ┌─────────▼─────────┐         ┌─────────▼─────────┐
                    │  Dependency Graph │         │    Plugin         │
                    │   Resolution      │         │   Existence       │
                    │                   │         │    Check          │
                    │ • Build Graph     │         │                   │
                    │ • Topological Sort│         │ • Check Registry  │
                    │ • Cycle Detection │         │ • Validate Name   │
                    │ • Filter Installed│         │                   │
                    └─────────┬─────────┘         └─────────┬─────────┘
                              │                             │
                              └───────────┬─────────────────┘
                                          │
                                          ▼
                                ┌─────────────────┐
                                │ Ordered Plugin  │
                                │  Installation   │
                                └─────┬─────┬─────┘
                                      │     │
                          ┌───────────▼     ▼───────────┐
                          │                             │
                ┌─────────▼─────────┐         ┌─────────▼─────────┐
                │     ArgoCD        │         │      Helm         │
                │   Deployment      │         │   Deployment      │
                └─────────┬─────────┘         └─────────┬─────────┘
                          │                             │
                          ▼                             ▼
                ┌─────────────────┐           ┌─────────────────┐
                │   Repository    │           │     Charts      │
                │      Pull       │           │      Pull       │
                └─────────┬───────┘           └─────────┬───────┘
                          │                             │
                          ▼                             │
                ┌─────────────────┐                     │
                │   ArgoCD App    │                     │
                │   Creation      │                     │
                └─────────┬───────┘                     │
                          │                             │
                          └─────────┬───────────────────┘
                                    │
                                    ▼
                          ┌─────────────────────────────────┐
                          │     Kubernetes Cluster         │
                          │                                 │
                          │  ┌─────────────────┐            │
                          │  │  K8s Plugin     │            │
                          │  │   Namespace     │            │
                          │  └─────────────────┘            │
                          │                                 │
                          │  ┌─────────────────┐            │
                          │  │ Installer       │            │
                          │  │   Tracker       │            │
                          │  └─────────────────┘            │
                          └─────────────────────────────────┘
```

### Plugin Removal Process

```
                                    ┌─────────────────┐
                                    │ Plugin Remove   │
                                    └─────────┬───────┘
                                              │
                                              ▼
                                    ┌─────────────────┐
                                    │DependencyValidator│
                                    │ValidateUninstall()│
                                    └─────┬─────┬─────┘
                                          │     │
                              ┌───────────▼     ▼─────────────┐
                              │                               │
                    ┌─────────▼─────────┐         ┌─────────▼─────────┐
                    │ Reverse Dependency│         │    Plugin         │
                    │   Validation      │         │   Existence       │
                    │                   │         │    Check          │
                    │ • Check dependents│         │                   │
                    │ • Prevent removal │         │ • Verify installed│
                    │   if others depend│         │ • Check registry  │
                    │ • Calculate order │         │                   │
                    └─────────┬─────────┘         └─────────┬─────────┘
                              │                             │
                              └───────────┬─────────────────┘
                                          │
                                          ▼
                                ┌─────────────────┐
                                │ Ordered Plugin  │
                                │   Removal       │
                                └─────┬─────┬─────┘
                                      │     │
                          ┌───────────▼     ▼───────────┐
                          │                             │
                ┌─────────▼─────────┐         ┌─────────▼─────────┐
                │   ArgoCD          │         │     Helm          │
                │   Installer       │         │   Installer       │
                └─────────┬─────────┘         └─────────┬─────────┘
                          │                             │
                          ▼                             ▼
                ┌─────────────────┐           ┌─────────────────┐
                │ Delete ArgoCD   │           │  Helm Removal   │
                │     App         │           │   Operation     │
                └─────────┬───────┘           └─────────┬───────┘
                          │                             │
                          └─────────┬───────────────────┘
                                    │
                                    ▼
                          ┌─────────────────────────────────┐
                          │     Kubernetes Cluster         │
                          │                                 │
                          │  ┌─────────────────┐            │
                          │  │  K8s Plugin     │            │
                          │  │   Namespace     │            │
                          │  │    Cleanup      │            │
                          │  └─────────────────┘            │
                          │                                 │
                          │  ┌─────────────────┐            │
                          │  │ Installer       │            │
                          │  │   Tracker       │            │
                          │  │    Update       │            │
                          │  └─────────────────┘            │
                          └─────────────────────────────────┘
```

### Dependency Resolution Examples

#### Safe Removal Example

```
┌──────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                              Scenario: Remove load-balancer                                              │
│                                                                                                          │
│  User Request: ./playground plugin remove --name load-balancer                                          │
│                                                                                                          │
│  ┌─────────────────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                            Dependency Validation                                                    │ │
│  │                                                                                                     │ │
│  │  Current State:                                                                                     │ │
│  │  • load-balancer: INSTALLED ✓                                                                      │ │
│  │  • nginx: INSTALLED ✓ (depends on load-balancer)                                                   │ │
│  │  • ingress: INSTALLED ✓ (depends on nginx, load-balancer)                                          │ │
│  │                                                                                                     │ │
│  │  Reverse Dependency Check:                                                                          │ │
│  │  • load-balancer has dependents: [nginx, ingress]                                                  │ │
│  │  • These plugins will break if load-balancer is removed                                            │ │
│  │                                                                                                     │ │
│  │  Result: ❌ BLOCK REMOVAL                                                                           │ │
│  │  Error: "Cannot remove load-balancer: required by nginx, ingress"                                  │ │
│  └─────────────────────────────────────────────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

#### Cascading Removal Example

```
┌──────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                           Scenario: Remove Multiple Plugins                                             │
│                                                                                                          │
│  User Request: ./playground plugin remove --name nginx,ingress,load-balancer                           │
│                                                                                                          │
│  ┌─────────────────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                            Dependency Resolution                                                    │ │
│  │                                                                                                     │ │
│  │  Target Plugins: [nginx, ingress, load-balancer]                                                   │ │
│  │  Dependencies Analysis:                                                                              │ │
│  │  • ingress depends on: [nginx, load-balancer]                                                      │ │
│  │  • nginx depends on: [load-balancer]                                                               │ │
│  │  • load-balancer depends on: []                                                                    │ │
│  │                                                                                                     │ │
│  │  Uninstall Order (Reverse Topological):                                                            │ │
│  │  1. ingress (remove first - depends on others)                                                     │ │
│  │  2. nginx (remove second - depends on load-balancer)                                               │ │
│  │  3. load-balancer (remove last - no dependencies)                                                  │ │
│  │                                                                                                     │ │
│  │  Result: ✅ PROCEED WITH ORDERED REMOVAL                                                           │ │
│  └─────────────────────────────────────────────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

### Dependency Inspection Command

```
┌──────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                               Plugin Dependencies Command                                                │
│                                                                                                          │
│  Command: ./playground plugin deps --cluster cluster-name                                               │
│                                                                                                          │
│  ┌─────────────────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                                   Output Example                                                    │ │
│  │                                                                                                     │ │
│  │  Plugin Dependency Information:                                                                     │ │
│  │                                                                                                     │ │
│  │  argocd:                                                                                            │ │
│  │    Dependencies: []                                                                                 │ │
│  │    Dependents: []                                                                                   │ │
│  │                                                                                                     │ │
│  │  cert-manager:                                                                                      │ │
│  │    Dependencies: []                                                                                 │ │
│  │    Dependents: [tls]                                                                                │ │
│  │                                                                                                     │ │
│  │  ingress:                                                                                           │ │
│  │    Dependencies: [nginx-ingress, load-balancer]                                                    │ │
│  │    Dependents: []                                                                                   │ │
│  │                                                                                                     │ │
│  │  load-balancer:                                                                                     │ │
│  │    Dependencies: []                                                                                 │ │
│  │    Dependents: [nginx-ingress, ingress]                                                            │ │
│  │                                                                                                     │ │
│  │  nginx-ingress:                                                                                     │ │
│  │    Dependencies: [load-balancer]                                                                    │ │
│  │    Dependents: [ingress]                                                                            │ │
│  │                                                                                                     │ │
│  │  tls:                                                                                               │ │
│  │    Dependencies: [cert-manager]                                                                     │ │
│  │    Dependents: []                                                                                   │ │
│  └─────────────────────────────────────────────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

## Core Components & Architecture

### Dependency Management System
- **DependencyGraph**: Unified graph structure with GraphNode containing plugin, dependencies, and dependents
- **DependencyValidator**: High-level validation orchestrator for install/uninstall operations
- **GraphNode**: Individual nodes containing plugin interface and bidirectional dependency relationships
- **Topological Sorting**: Algorithms for dependency resolution and safe removal ordering
- **Cycle Detection**: DFS-based circular dependency detection with stack tracking

### Plugin System
- **DependencyPlugin Interface**: Extended plugin interface with `GetDependencies() []string`
- **Plugin Registry**: Available plugins with their dependency definitions
- **Installation Orchestration**: Order-aware plugin installation with dependency resolution
- **Removal Validation**: Reverse dependency checking to prevent breaking changes

## Technology Stack

- **GoLang**: Core application development
- **Kubernetes APIs**: Cluster management and plugin deployment
- **ArgoCD**: GitOps-based application deployment
- **Helm**: Package management for Kubernetes applications
- **MetalLB**: Load balancer implementation
- **Multipass**: VM orchestration and management
- **Graph Algorithms**: Topological sorting and cycle detection for dependency resolution 
