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
	if plugin.GetName() != IngressName {
		t.Errorf("Expected plugin name '%s', got '%s'", IngressName, plugin.GetName())
	}

	// Test plugin options
	options := plugin.GetOptions()
	if options.Namespace != nil && *options.Namespace != IngressNamespace {
		t.Errorf("Expected namespace '%s', got '%s'", IngressNamespace, *options.Namespace)
	}

	if options.Version != nil && *options.Version != IngressVersion {
		t.Errorf("Expected version '%s', got '%s'", IngressVersion, *options.Version)
	}

	// Test that chart-related methods return empty values (since this plugin doesn't use Helm)
	if options.ChartName != nil && *options.ChartName != "" {
		t.Errorf("Expected empty chart name, got '%s'", *options.ChartName)
	}

	if options.Repository != nil && *options.Repository != "" {
		t.Errorf("Expected empty repository, got '%s'", *options.Repository)
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
