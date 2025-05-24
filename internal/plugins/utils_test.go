package plugins

import (
	"testing"

	"github.com/mrgb7/playground/internal/installer"
)

func TestIsArgoCDRunning(t *testing.T) {
	tests := []struct {
		name       string
		kubeConfig string
		expected   bool
	}{
		{
			name:       "invalid kubeconfig",
			kubeConfig: "invalid-config",
			expected:   false,
		},
		{
			name:       "empty kubeconfig",
			kubeConfig: "",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsArgoCDRunning(tt.kubeConfig)
			if result != tt.expected {
				t.Errorf("IsArgoCDRunning() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestNewArgoOptions(t *testing.T) {
	tests := []struct {
		name         string
		pluginName   string
		expectedApp  string
		expectedRepo string
		expectedPath string
		expectedNS   string
	}{
		{
			name:         "cert-manager plugin",
			pluginName:   "cert-manager",
			expectedApp:  "cert-manager-app",
			expectedRepo: "https://github.com/mrgb7/core-infrastructure",
			expectedPath: "cert-manager",
			expectedNS:   "cert-manager",
		},
		{
			name:         "argocd plugin",
			pluginName:   "argocd",
			expectedApp:  "argocd-app",
			expectedRepo: "https://github.com/mrgb7/core-infrastructure",
			expectedPath: "argocd",
			expectedNS:   "argocd",
		},
		{
			name:         "loadBalancer plugin",
			pluginName:   "loadBalancer",
			expectedApp:  "metallb-app",
			expectedRepo: "https://github.com/mrgb7/core-infrastructure",
			expectedPath: "metallb",
			expectedNS:   "metallb-system",
		},
		{
			name:         "nginx plugin",
			pluginName:   "nginx",
			expectedApp:  "nginx-app",
			expectedRepo: "https://github.com/mrgb7/core-infrastructure",
			expectedPath: "nginx",
			expectedNS:   "nginx-system",
		},
		{
			name:         "unknown plugin",
			pluginName:   "unknown",
			expectedApp:  "unknown-app",
			expectedRepo: "https://github.com/mrgb7/core-infrastructure",
			expectedPath: "unknown",
			expectedNS:   "unknown-system",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockPlugin{name: tt.pluginName}
			opts := NewArgoOptions(mock)

			if opts.ApplicationName != tt.expectedApp {
				t.Errorf("ApplicationName = %v, expected %v", opts.ApplicationName, tt.expectedApp)
			}

			if opts.RepoURL != tt.expectedRepo {
				t.Errorf("RepoURL = %v, expected %v", opts.RepoURL, tt.expectedRepo)
			}

			if opts.Path != tt.expectedPath {
				t.Errorf("Path = %v, expected %v", opts.Path, tt.expectedPath)
			}

			if opts.Namespace != tt.expectedNS {
				t.Errorf("Namespace = %v, expected %v", opts.Namespace, tt.expectedNS)
			}

			if opts.TargetRevision != "main" {
				t.Errorf("TargetRevision = %v, expected 'main'", opts.TargetRevision)
			}
		})
	}
}

func TestNewInstaller(t *testing.T) {
	tests := []struct {
		name         string
		kubeConfig   string
		clusterName  string
		expectError  bool
		pluginName   string
	}{
		{
			name:        "invalid kubeconfig",
			kubeConfig:  "invalid-config",
			clusterName: "test-cluster",
			expectError: false,
			pluginName:  "test-plugin",
		},
		{
			name:        "empty cluster name",
			kubeConfig:  createValidKubeConfig(),
			clusterName: "",
			expectError: false,
			pluginName:  "test-plugin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockPlugin{name: tt.pluginName}
			inst, err := NewInstaller(mock, tt.kubeConfig, tt.clusterName)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError && inst == nil {
				t.Errorf("Expected installer but got nil")
			}
		})
	}
}

type MockPlugin struct {
	name string
}

func (m *MockPlugin) GetName() string {
	return m.name
}

func (m *MockPlugin) GetInstaller() (installer.Installer, error) {
	return &MockInstaller{}, nil
}

func (m *MockPlugin) Install(ensure ...bool) error {
	return nil
}

func (m *MockPlugin) Uninstall(ensure ...bool) error {
	return nil
}

func (m *MockPlugin) Status() string {
	return "mock status"
}

type MockInstaller struct{}

func (m *MockInstaller) Install(options *installer.InstallOptions) error {
	return nil
}

func (m *MockInstaller) UnInstall(options *installer.InstallOptions) error {
	return nil
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