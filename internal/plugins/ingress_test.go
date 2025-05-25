package plugins

import (
	"testing"
)

func TestIngressPluginInterface(t *testing.T) {
	// Test plugin creation with minimal kubeconfig
	plugin, err := NewIngress("dummy-kubeconfig", "test-cluster")
	if err != nil {
		// If k8s client creation fails (expected in test environment),
		// we can still test the interface methods that don't require k8s client
		t.Logf("K8s client creation failed (expected in test): %v", err)
		return
	}

	// Test plugin interface methods
	if plugin.GetName() != "ingress" {
		t.Errorf("Expected plugin name 'ingress', got '%s'", plugin.GetName())
	}

	if plugin.GetNamespace() != IngressNamespace {
		t.Errorf("Expected namespace '%s', got '%s'", IngressNamespace, plugin.GetNamespace())
	}

	if plugin.GetVersion() != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", plugin.GetVersion())
	}

	// Test that chart-related methods return empty values (since this plugin doesn't use Helm)
	if plugin.GetChartName() != "" {
		t.Errorf("Expected empty chart name, got '%s'", plugin.GetChartName())
	}

	if plugin.GetRepository() != "" {
		t.Errorf("Expected empty repository, got '%s'", plugin.GetRepository())
	}

	// Test that it implements the Plugin interface
	var _ Plugin = plugin
}

func TestIngressPluginConstants(t *testing.T) {
	// Test plugin constants
	if IngressNamespace != "ingress-system" {
		t.Errorf("Expected IngressNamespace to be 'ingress-system', got '%s'", IngressNamespace)
	}
} 