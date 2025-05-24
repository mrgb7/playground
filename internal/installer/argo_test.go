package installer

import (
	"context"
	"testing"
	"time"
)

func TestNewArgoInstaller(t *testing.T) {
	tests := []struct {
		name        string
		kubeConfig  string
		clusterName string
		expectError bool
	}{
		{
			name:        "invalid config",
			kubeConfig:  "invalid-config",
			clusterName: "test-cluster",
			expectError: true,
		},
		{
			name:        "empty config",
			kubeConfig:  "",
			clusterName: "test-cluster",
			expectError: true,
		},
		{
			name:        "empty cluster name",
			kubeConfig:  createValidKubeConfig(),
			clusterName: "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installer, err := NewArgoInstaller(tt.kubeConfig, tt.clusterName)
			
			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			
			if !tt.expectError && installer != nil {
				if installer.ClusterName != tt.clusterName {
					t.Errorf("expected cluster name %s, got %s", tt.clusterName, installer.ClusterName)
				}
				
				if installer.ArgoNamespace != DefaultArgoNamespace {
					t.Errorf("expected namespace %s, got %s", DefaultArgoNamespace, installer.ArgoNamespace)
				}
				
				if installer.LocalPort != DefaultLocalPort {
					t.Errorf("expected local port %d, got %d", DefaultLocalPort, installer.LocalPort)
				}
			}
		})
	}
}

func TestArgoInstaller_ValidateArgoConnection(t *testing.T) {
	tests := []struct {
		name          string
		serverAddress string
		expectError   bool
	}{
		{
			name:          "valid connection",
			serverAddress: "localhost:8080",
			expectError:   false,
		},
		{
			name:          "no connection",
			serverAddress: "",
			expectError:   true,
		},
		{
			name:          "invalid address format",
			serverAddress: "invalid-address",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installer := &ArgoInstaller{
				ServerAddress: tt.serverAddress,
			}

			err := installer.ValidateArgoConnection()
			
			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestArgoInstaller_closePortForward(t *testing.T) {
	installer := &ArgoInstaller{}
	
	ctx, cancel := context.WithCancel(context.Background())
	installer.portForwardCancel = cancel
	
	installer.closePortForward()
	
	select {
	case <-ctx.Done():
	case <-time.After(100 * time.Millisecond):
		t.Errorf("context was not cancelled")
	}
	
	if installer.portForwardCancel != nil {
		t.Errorf("portForwardCancel should be nil after close")
	}
}

func TestArgoInstaller_closePortForward_NilCancel(t *testing.T) {
	installer := &ArgoInstaller{}
	installer.portForwardCancel = nil
	
	installer.closePortForward()
}

func TestInstallOptions_Validation(t *testing.T) {
	tests := []struct {
		name    string
		options *InstallOptions
		valid   bool
	}{
		{
			name: "valid options",
			options: &InstallOptions{
				ApplicationName: "test-app",
				RepoURL:        "https://github.com/test/repo",
				Path:           "manifests/",
				TargetRevision: "main",
				Namespace:      "test-namespace",
			},
			valid: true,
		},
		{
			name: "missing application name",
			options: &InstallOptions{
				RepoURL:        "https://github.com/test/repo",
				Path:           "manifests/",
				TargetRevision: "main",
				Namespace:      "test-namespace",
			},
			valid: false,
		},
		{
			name: "missing repo URL",
			options: &InstallOptions{
				ApplicationName: "test-app",
				Path:           "manifests/",
				TargetRevision: "main",
				Namespace:      "test-namespace",
			},
			valid: false,
		},
		{
			name: "nil options",
			options: nil,
			valid: false,
		},
		{
			name: "empty strings",
			options: &InstallOptions{
				ApplicationName: "",
				RepoURL:        "",
				Path:           "",
				TargetRevision: "",
				Namespace:      "",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := validateInstallOptions(tt.options)
			if valid != tt.valid {
				t.Errorf("expected validation result %v, got %v", tt.valid, valid)
			}
		})
	}
}

func TestArgoInstaller_DefaultValues(t *testing.T) {
	installer := &ArgoInstaller{
		ArgoNamespace:  DefaultArgoNamespace,
		ArgoServerPort: DefaultArgoServerPort,
		LocalPort:      DefaultLocalPort,
	}
	
	if installer.ArgoNamespace != "argocd" {
		t.Errorf("expected default namespace 'argocd', got '%s'", installer.ArgoNamespace)
	}
	
	if installer.ArgoServerPort != 443 {
		t.Errorf("expected default server port 443, got %d", installer.ArgoServerPort)
	}
	
	if installer.LocalPort != 8080 {
		t.Errorf("expected default local port 8080, got %d", installer.LocalPort)
	}
}

func TestArgoInstaller_Install_NilOptions(t *testing.T) {
	installer := &ArgoInstaller{}
	
	err := installer.Install(nil)
	if err == nil {
		t.Errorf("expected error with nil options, but got none")
	}
}

func TestArgoInstaller_UnInstall_NilOptions(t *testing.T) {
	installer := &ArgoInstaller{}
	
	err := installer.UnInstall(nil)
	if err == nil {
		t.Errorf("expected error with nil options, but got none")
	}
}

func TestArgoInstaller_StructInitialization(t *testing.T) {
	tests := []struct {
		name      string
		installer *ArgoInstaller
		expected  map[string]interface{}
	}{
		{
			name: "default initialization",
			installer: &ArgoInstaller{
				ArgoNamespace:  DefaultArgoNamespace,
				ArgoServerPort: DefaultArgoServerPort,
				LocalPort:      DefaultLocalPort,
			},
			expected: map[string]interface{}{
				"namespace":   "argocd",
				"serverPort":  443,
				"localPort":   8080,
			},
		},
		{
			name: "custom initialization",
			installer: &ArgoInstaller{
				KubeConfig:     "custom-config",
				ClusterName:    "custom-cluster",
				ArgoNamespace:  "custom-argocd",
				ArgoServerPort: 8443,
				LocalPort:      9090,
				ServerAddress:  "localhost:9090",
			},
			expected: map[string]interface{}{
				"kubeConfig":     "custom-config",
				"clusterName":    "custom-cluster",
				"namespace":      "custom-argocd",
				"serverPort":     8443,
				"localPort":      9090,
				"serverAddress":  "localhost:9090",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.installer.ArgoNamespace != tt.expected["namespace"] {
				t.Errorf("expected namespace %v, got %v", tt.expected["namespace"], tt.installer.ArgoNamespace)
			}
			
			if tt.installer.ArgoServerPort != tt.expected["serverPort"] {
				t.Errorf("expected server port %v, got %v", tt.expected["serverPort"], tt.installer.ArgoServerPort)
			}
			
			if tt.installer.LocalPort != tt.expected["localPort"] {
				t.Errorf("expected local port %v, got %v", tt.expected["localPort"], tt.installer.LocalPort)
			}
			
			if expectedConfig, ok := tt.expected["kubeConfig"]; ok {
				if tt.installer.KubeConfig != expectedConfig {
					t.Errorf("expected kubeConfig %v, got %v", expectedConfig, tt.installer.KubeConfig)
				}
			}
			
			if expectedCluster, ok := tt.expected["clusterName"]; ok {
				if tt.installer.ClusterName != expectedCluster {
					t.Errorf("expected clusterName %v, got %v", expectedCluster, tt.installer.ClusterName)
				}
			}
			
			if expectedAddress, ok := tt.expected["serverAddress"]; ok {
				if tt.installer.ServerAddress != expectedAddress {
					t.Errorf("expected serverAddress %v, got %v", expectedAddress, tt.installer.ServerAddress)
				}
			}
		})
	}
}

func TestArgoInstaller_Constants(t *testing.T) {
	if DefaultArgoNamespace != "argocd" {
		t.Errorf("expected DefaultArgoNamespace to be 'argocd', got '%s'", DefaultArgoNamespace)
	}
	
	if DefaultArgoServerPort != 443 {
		t.Errorf("expected DefaultArgoServerPort to be 443, got %d", DefaultArgoServerPort)
	}
	
	if DefaultLocalPort != 8080 {
		t.Errorf("expected DefaultLocalPort to be 8080, got %d", DefaultLocalPort)
	}
}

func TestInstallOptions_ComplexValidation(t *testing.T) {
	tests := []struct {
		name    string
		options *InstallOptions
		valid   bool
	}{
		{
			name: "valid with all fields",
			options: &InstallOptions{
				ApplicationName: "complex-app",
				RepoURL:        "https://github.com/argoproj/argocd-example-apps",
				Path:           "guestbook",
				TargetRevision: "HEAD",
				Namespace:      "guestbook",
				Values: map[string]interface{}{
					"image.tag":      "latest",
					"replicaCount":   3,
					"service.type":   "LoadBalancer",
				},
			},
			valid: true,
		},
		{
			name: "valid minimal required fields",
			options: &InstallOptions{
				ApplicationName: "minimal-app",
				RepoURL:        "https://github.com/test/repo",
			},
			valid: true,
		},
		{
			name: "valid with special characters",
			options: &InstallOptions{
				ApplicationName: "app-with-special_chars.123",
				RepoURL:        "https://github.com/org/repo-name_with.special-chars",
				Path:           "charts/app-chart",
				TargetRevision: "v1.2.3-beta.1",
				Namespace:      "namespace-with-dashes",
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := validateInstallOptions(tt.options)
			if valid != tt.valid {
				t.Errorf("expected validation result %v, got %v", tt.valid, valid)
			}
		})
	}
}

func createValidKubeConfig() string {
	return `
apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://kubernetes.docker.internal:6443
    insecure-skip-tls-verify: true
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: test-context
current-context: test-context
users:
- name: test-user
  user:
    token: test-token
`
}

func validateInstallOptions(options *InstallOptions) bool {
	if options == nil {
		return false
	}
	
	if options.ApplicationName == "" {
		return false
	}
	
	if options.RepoURL == "" {
		return false
	}
	
	return true
} 