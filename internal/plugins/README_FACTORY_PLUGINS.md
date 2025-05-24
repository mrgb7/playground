# Factory-Based Plugin Installation System

This document describes the factory-based plugin installation system that automatically detects if ArgoCD is running in the cluster and chooses the appropriate installer (ArgoCD or Helm) accordingly.

## Overview

The factory-based plugin system provides intelligent installation management by:
- **Auto-detecting ArgoCD**: Checks if ArgoCD is running and ready in the cluster
- **Adaptive Installation**: Uses ArgoCD installer when available, falls back to Helm
- **Unified Interface**: Provides a consistent experience regardless of the underlying installer
- **Graceful Fallback**: Automatically falls back to Helm if ArgoCD installation fails
- **GitOps Integration**: Creates and manages ArgoCD applications via REST API when ArgoCD is available

## Architecture

### Components

1. **Installer Factory** (`utils.go`)
   - `IsArgoCDRunning()`: Detects if ArgoCD is operational
   - `NewInstaller()`: Returns appropriate installer based on ArgoCD status
   - `NewArgoOptions()`: Creates ArgoCD application specifications

2. **Base Plugin** (`base.go`)
   - `BasePlugin`: Provides factory-based installation capabilities
   - `FactoryInstall()`: Factory-based installation method
   - `FactoryUninstall()`: Factory-based uninstallation method

3. **ArgoCD Installer** (`argo.go`)
   - **REST API Integration**: Communicates with ArgoCD server via HTTP REST API
   - **Authentication**: Uses ArgoCD admin credentials for API access
   - **Port Forwarding**: Sets up secure port forwarding to ArgoCD server
   - **Application Management**: Creates and deletes ArgoCD applications programmatically

4. **Plugin Interface** (`plugin.go`)
   - `Factory`: Extended interface for factory-aware plugins
   - Backward compatibility with existing `Plugin` interface

### ArgoCD REST API Integration

The ArgoCD installer now includes complete REST API integration for managing applications:

#### Authentication Flow
1. **Port Forward Setup**: Establishes secure connection to ArgoCD server pod
2. **Credential Retrieval**: Fetches admin password from Kubernetes secret
3. **Session Creation**: Authenticates with ArgoCD API to obtain JWT token
4. **Authenticated Requests**: Uses Bearer token for all subsequent API calls

#### Application Lifecycle Management
- **Create Applications**: POST to `/api/v1/applications` with full application specification
- **Delete Applications**: DELETE from `/api/v1/applications/{name}` with cascade deletion
- **Automatic Sync**: Applications configured with automated sync policies
- **GitOps Integration**: Applications reference Git repositories for source-of-truth

### Plugin Enhancement

Each plugin now embeds `BasePlugin` to gain factory-based installation capabilities:

```go
type CertManager struct {
    KubeConfig string
    *BasePlugin  // Embedded for factory functionality
}

func NewCertManager(kubeConfig string) *CertManager {
    cm := &CertManager{KubeConfig: kubeConfig}
    cm.BasePlugin = NewBasePlugin(kubeConfig, cm)
    return cm
}
```

## ArgoCD Detection Logic

The system uses multiple checks to determine if ArgoCD is ready:

1. **Namespace Verification**: Checks if `argocd` namespace exists
2. **Pod Discovery**: Finds ArgoCD server pods using label selector
3. **Readiness Check**: Verifies pods are running and all containers are ready
4. **Timeout Handling**: Uses 10-second timeout for detection operations

```go
func IsArgoCDRunning(kubeConfig string) bool {
    // 1. Create k8s client
    // 2. Check namespace exists
    // 3. List ArgoCD server pods
    // 4. Verify pod readiness
    // 5. Return true if fully operational
}
```

## Installation Flow

### Factory-Based Installation Process

```
Plugin Installation Request
           ↓
    Check ArgoCD Status
           ↓
    ┌─────────────────┐    ┌─────────────────┐
    │  ArgoCD Running │    │ ArgoCD Not Found│
    │       ↓         │    │       ↓         │
    │ Use ArgoInstaller│    │ Use HelmInstaller│
    │       ↓         │    │       ↓         │
    │ Create ArgoApp  │    │ Deploy Helm Chart│
    └─────────────────┘    └─────────────────┘
           ↓                        ↓
    ┌─────────────────────────────────────────┐
    │          Installation Complete          │
    └─────────────────────────────────────────┘
```

### ArgoCD Application Creation Flow

When ArgoCD is detected, the following process occurs:

```
1. Setup Port Forward → ArgoCD Server
           ↓
2. Authenticate → Get JWT Token  
           ↓
3. Create Application Spec → JSON Payload
           ↓
4. POST /api/v1/applications → ArgoCD API
           ↓
5. Application Created → GitOps Management
```

### ArgoCD Application Configuration

When ArgoCD is detected, plugins are deployed as ArgoCD applications with full specifications:

```go
&installer.InstallOptions{
    ApplicationName: "cert-manager-app",
    RepoURL:        "https://github.com/mrgb7/core-infrastructure",
    Path:           "cert-manager",
    TargetRevision: "main",
    Namespace:      "cert-manager",
}
```

This creates a complete ArgoCD Application resource:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: cert-manager-app
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/mrgb7/core-infrastructure
    path: cert-manager
    targetRevision: main
  destination:
    server: https://kubernetes.default.svc
    namespace: cert-manager
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

## Plugin Mappings

Each plugin has predefined ArgoCD application configurations:

| Plugin | Application Name | Repository Path | Namespace | Auto-Sync | Self-Heal |
|--------|------------------|-----------------|-----------|-----------|-----------|
| cert-manager | cert-manager-app | cert-manager | cert-manager | ✓ | ✓ |
| argocd | argocd-app | argocd | argocd | ✓ | ✓ |
| loadBalancer | metallb-app | metallb | metallb-system | ✓ | ✓ |
| nginx | nginx-app | nginx | nginx-system | ✓ | ✓ |

## Usage

### CLI Commands

The existing CLI commands automatically use factory-based installation:

```bash
# Factory-based installation - uses ArgoCD if available, otherwise Helm
playground cluster plugin add --name cert-manager --cluster my-cluster

# Factory-based removal - uses same installer as installation
playground cluster plugin remove --name cert-manager --cluster my-cluster
```

### Programmatic Usage

```go
// Create plugin with factory capabilities
cm := plugins.NewCertManager(kubeConfig)

// Use factory-based installation
err := cm.FactoryInstall(kubeConfig, clusterName)

// Use factory-based uninstallation  
err := cm.FactoryUninstall(kubeConfig, clusterName)
```

## Benefits

### 1. GitOps Integration
When ArgoCD is available, plugins are managed as GitOps applications:
- **Version Control**: All configurations stored in Git repository
- **Audit Trail**: Complete change history and deployment tracking
- **Rollback Capability**: Easy reversion to previous versions using Git history
- **Drift Detection**: Automatic detection and correction of configuration drift
- **Declarative Management**: Infrastructure as Code principles applied

### 2. Operational Excellence
- **Self-Healing**: ArgoCD applications automatically sync with desired state
- **Compliance**: GitOps ensures configuration compliance and governance
- **Scalability**: ArgoCD handles large-scale deployments efficiently
- **Observability**: Built-in monitoring and alerting for application health

### 3. Developer Experience
- **Unified Interface**: Same commands work for both installation methods
- **Consistent Experience**: Transparent to end users regardless of installer
- **Automatic Optimization**: Always uses the best available installer
- **Seamless Fallback**: No manual intervention required when ArgoCD is unavailable

### 4. Security and Reliability
- **Secure Communication**: TLS-encrypted communication with ArgoCD API
- **Token-based Authentication**: JWT tokens for secure API access
- **Port Forwarding**: Secure tunnel to ArgoCD server without exposing services
- **Error Handling**: Comprehensive error handling and graceful degradation

## Technical Implementation

### ArgoCD API Integration

The ArgoCD installer implements full REST API integration:

```go
// Authentication with ArgoCD
func (a *ArgoInstaller) authenticate() error {
    // 1. Get admin password from Kubernetes secret
    // 2. Create session request with credentials
    // 3. POST to /api/v1/session
    // 4. Extract JWT token from response
    // 5. Store token for subsequent requests
}

// Application creation
func (a *ArgoInstaller) createApplication(options *InstallOptions) error {
    // 1. Build ArgoCD Application specification
    // 2. Marshal to JSON payload
    // 3. POST to /api/v1/applications
    // 4. Handle response and error cases
}

// Application deletion
func (a *ArgoInstaller) deleteApplication(options *InstallOptions) error {
    // 1. Create DELETE request with cascade parameter
    // 2. DELETE from /api/v1/applications/{name}
    // 3. Handle response codes (200, 204, 404)
}
```

### Error Handling and Resilience

- **Connection Failures**: Graceful fallback to Helm installer
- **Authentication Errors**: Proper error reporting and debugging information
- **API Timeouts**: Configurable timeouts for all HTTP operations
- **Network Issues**: Retry logic and connection pooling
- **Port Forward Cleanup**: Automatic cleanup of resources on completion

## Error Handling

The system provides comprehensive error handling:

1. **Detection Failures**: Falls back to Helm if ArgoCD detection fails
2. **Installation Failures**: Attempts fallback installation method
3. **API Failures**: Detailed error reporting for troubleshooting
4. **Authentication Issues**: Clear error messages for credential problems
5. **Timeout Handling**: Uses appropriate timeouts for all operations

## Troubleshooting

### Common Issues and Solutions

#### ArgoCD Not Detected
- **Symptom**: Plugins install via Helm instead of ArgoCD
- **Causes**: ArgoCD pods not ready, namespace missing, network issues
- **Solution**: Check ArgoCD installation status, verify namespace exists

#### Authentication Failures
- **Symptom**: "failed to authenticate with ArgoCD" errors
- **Causes**: Incorrect admin password, secret not found, API unreachable
- **Solution**: Verify ArgoCD installation, check admin secret exists

#### Application Creation Failures
- **Symptom**: Applications not appearing in ArgoCD UI
- **Causes**: Invalid repository URL, path not found, permission issues
- **Solution**: Verify Git repository accessibility, check application logs

#### Port Forward Issues
- **Symptom**: "failed to setup port forward" errors
- **Causes**: ArgoCD server pod not running, network policies, firewall
- **Solution**: Check pod status, verify network connectivity

### Debug Mode

Enable debug logging for detailed troubleshooting:

```bash
export LOG_LEVEL=debug
playground cluster plugin add --name cert-manager --cluster my-cluster
```

### Force Helm Installation

To bypass ArgoCD detection and force Helm installation:

```go
// Use regular installation to bypass factory detection
err := plugin.Install()
```

## Contributing

When adding new plugins:

1. Implement the `Factory` interface
2. Embed `BasePlugin` for factory capabilities
3. Add mapping in `NewArgoOptions()`
4. Create comprehensive tests including API integration
5. Update documentation with new plugin details
6. Ensure proper error handling for all scenarios

## Testing

The system includes comprehensive test coverage:

- **Unit Tests**: Core functionality and error handling
- **Integration Tests**: ArgoCD API interactions
- **Mock Testing**: HTTP client behavior and network failures
- **Edge Cases**: Nil parameters, timeout scenarios, authentication failures

Run tests with:

```bash
go test ./internal/installer/ -v
go test ./internal/plugins/ -v
``` 