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

func TestNewInstaller(t *testing.T) {
	tests := []struct {
		name        string
		kubeConfig  string
		clusterName string
		expectError bool
		pluginName  string
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

func (m *MockPlugin) Install(kubeConfig, clusterName string, ensure ...bool) error {
	return nil
}

func (m *MockPlugin) Uninstall(kubeConfig, clusterName string, ensure ...bool) error {
	return nil
}

func (m *MockPlugin) Status() string {
	return "mock status"
}

func (m *MockPlugin) GetNamespace() string {
	return "test-namespace"
}

func (m *MockPlugin) GetVersion() string {
	return "1.0.0"
}

func (m *MockPlugin) GetChartName() string {
	return "test-chart"
}

func (m *MockPlugin) GetRepository() string {
	return "https://test.repo.com"
}

func (m *MockPlugin) GetChartValues() map[string]interface{} {
	return map[string]interface{}{"test": "value"}
}

func (m *MockPlugin) GetRepoName() string {
	return "test-repo"
}

func (m *MockPlugin) OwnsNamespace() bool {
	return true
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
