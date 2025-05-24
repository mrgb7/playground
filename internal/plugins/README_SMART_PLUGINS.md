# Factory-Based Plugin Installation System

This document describes the factory-based plugin installation system that automatically detects if ArgoCD is running in the cluster and chooses the appropriate installer (ArgoCD or Helm) accordingly.

## Overview

The factory-based plugin system provides intelligent installation management by:
- **Auto-detecting ArgoCD**: Checks if ArgoCD is running and ready in the cluster
- **Adaptive Installation**: Uses ArgoCD installer when available, falls back to Helm
- **Unified Interface**: Provides a consistent experience regardless of the underlying installer
- **Graceful Fallback**: Automatically falls back to Helm if ArgoCD installation fails

## Architecture

### Components

1. **Installer Factory** (`utils.go`)
   - `IsArgoCDRunning()`: Detects if ArgoCD is operational
   - `CreateInstaller()`: Returns appropriate installer based on ArgoCD status
   - `CreateArgoInstallOptions()`: Creates ArgoCD application specifications

2. **Base Plugin** (`base.go`)
   - `BasePlugin`: Provides factory-based installation capabilities
   - `InstallWithFactory()`: Factory-based installation method
   - `UninstallWithFactory()`: Factory-based uninstallation method

3. **Plugin Interface** (`plugin.go`)
   - `FactoryAwarePlugin`: Extended interface for factory-aware plugins
   - Backward compatibility with existing `Plugin` interface

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

### ArgoCD Application Configuration

When ArgoCD is detected, plugins are deployed as ArgoCD applications:

```go
&installer.InstallOptions{
    ApplicationName: "cert-manager-app",
    RepoURL:        "https://github.com/mrgb7/core-infrastructure",
    Path:           "cert-manager",
    TargetRevision: "main",
    Namespace:      "cert-manager",
}
```

## Plugin Mappings

Each plugin has predefined ArgoCD application configurations:

| Plugin | Application Name | Repository Path | Namespace |
|--------|------------------|-----------------|-----------|
| cert-manager | cert-manager-app | cert-manager | cert-manager |
| argocd | argocd-app | argocd | argocd |
| loadBalancer | metallb-app | metallb | metallb-system |
| nginx | nginx-app | nginx | nginx-system |

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
err := cm.InstallWithFactory(kubeConfig, clusterName)

// Use factory-based uninstallation  
err := cm.UninstallWithFactory(kubeConfig, clusterName)
```

## Benefits

### 1. GitOps Integration
When ArgoCD is available, plugins are managed as GitOps applications:
- **Version Control**: All configurations stored in Git
- **Audit Trail**: Complete change history
- **Rollback Capability**: Easy reversion to previous versions
- **Drift Detection**: Automatic detection of configuration drift

### 2. Unified Management
- **Single Interface**: Same commands work for both installation methods
- **Consistent Experience**: Transparent to end users
- **Automatic Optimization**: Always uses the best available installer

### 3. Operational Excellence
- **Self-Healing**: ArgoCD applications automatically sync
- **Compliance**: GitOps ensures configuration compliance
- **Scalability**: ArgoCD handles large-scale deployments efficiently

## Error Handling

The system provides comprehensive error handling:

1. **Detection Failures**: Falls back to Helm if ArgoCD detection fails
2. **Installation Failures**: Attempts fallback installation method
3. **Timeout Handling**: Uses appropriate timeouts for all operations
4. **Logging**: Comprehensive logging for troubleshooting

```go
if factoryAwarePlugin, ok := plugin.(plugins.FactoryAwarePlugin); ok {
    err := factoryAwarePlugin.InstallWithFactory(kubeConfig, clusterName)
    if err != nil {
        // Fallback to regular installation
        err = plugin.Install()
    }
}
```

## Configuration

### ArgoCD Repository Settings

The system uses a centralized infrastructure repository:
- **Repository**: `https://github.com/mrgb7/core-infrastructure`
- **Branch**: `main`
- **Structure**: Each plugin has its own directory with manifests

### Detection Parameters

```go
const (
    ArgocdInstallNamespace    = "argocd"
    ArgocdServerLabelSelector = "app.kubernetes.io/name=argocd-server"
    DetectionTimeout          = 10 * time.Second
)
```

## Testing

The system includes comprehensive tests:

- **Unit Tests**: Test individual components
- **Integration Tests**: Test end-to-end workflows  
- **Mock Objects**: Simulate various scenarios
- **Edge Cases**: Handle error conditions

```bash
# Run plugin tests
go test ./internal/plugins/

# Run with verbose output
go test -v ./internal/plugins/
```

## Future Enhancements

Potential improvements:

1. **Multi-Repository Support**: Support for multiple Git repositories
2. **Custom Detection Logic**: Plugin-specific detection rules
3. **Health Monitoring**: Continuous health checks for installed plugins
4. **Dependency Management**: Handle plugin dependencies automatically
5. **Backup/Restore**: Automatic backup before major changes

## Troubleshooting

### Common Issues

1. **ArgoCD Not Detected**: Check namespace and pod labels
2. **Permission Errors**: Verify RBAC permissions for ArgoCD
3. **Repository Access**: Ensure Git repository is accessible
4. **Network Issues**: Check connectivity to ArgoCD server

### Debug Mode

Enable debug logging for detailed information:

```bash
export LOG_LEVEL=debug
playground cluster plugin add --name cert-manager --cluster my-cluster
```

### Manual Override

Force Helm installation even if ArgoCD is available:

```go
// Use regular installation to bypass smart detection
err := plugin.Install()
```

## Contributing

When adding new plugins:

1. Implement the `SmartPlugin` interface
2. Embed `BasePlugin` for smart capabilities
3. Add mapping in `CreateArgoInstallOptions()`
4. Create comprehensive tests
5. Update documentation 