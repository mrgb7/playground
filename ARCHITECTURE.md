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

## Plugin Management Flow

### Plugin Installation Process

```
                                    ┌─────────────────┐
                                    │   Add Plugin    │
                                    └─────────┬───────┘
                                              │
                                              ▼
                                    ┌─────────────────┐
                                    │    Validate     │
                                    └─────┬─────┬─────┘
                                          │     │
                              ┌───────────▼     ▼─────────────┐
                              │                               │
                    ┌─────────▼─────────┐         ┌─────────▼─────────┐
                    │   Dependencies    │         │    Existence      │
                    │     Check         │         │     Check         │
                    └─────────┬─────────┘         └─────────┬─────────┘
                              │                             │
                              └───────────┬─────────────────┘
                                          │
                                          ▼
                                ┌─────────────────┐
                                │ Installer       │
                                │ Factory         │
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
                                    │    Validate     │
                                    └─────┬─────┬─────┘
                                          │     │
                              ┌───────────▼     ▼─────────────┐
                              │                               │
                    ┌─────────▼─────────┐         ┌─────────▼─────────┐
                    │   Dependencies    │         │    Existence      │
                    │     Check         │         │     Check         │
                    └─────────┬─────────┘         └─────────┬─────────┘
                              │                             │
                              └───────────┬─────────────────┘
                                          │
                                          ▼
                                ┌─────────────────┐
                                │ Get Installer   │
                                │ from Tracker    │
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

## Technology Stack

- GoLang
- K8s and its APIs
- ArgoCD
- Helm
- MetalLB
- Multipass 
