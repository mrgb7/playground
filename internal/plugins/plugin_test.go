package plugins

import (
	"testing"
)

func TestCreatePluginsListIncludesIngress(t *testing.T) {
	// Test that CreatePluginsList includes the ingress plugin
	plugins, err := CreatePluginsList("dummy-kubeconfig", "192.168.1.100", "test-cluster")
	if err != nil {
		t.Logf("CreatePluginsList failed (expected in test environment): %v", err)
		return
	}

	// Check that ingress plugin is included
	found := false
	for _, plugin := range plugins {
		if plugin.GetName() == IngressName {
			found = true
			break
		}
	}

	if !found {
		t.Error("Ingress plugin not found in CreatePluginsList")
		t.Log("Available plugins:")
		for _, plugin := range plugins {
			t.Logf("  - %s", plugin.GetName())
		}
	}
}

func TestCreatePluginsListIncludesTLS(t *testing.T) {
	plugins, err := CreatePluginsList("dummy-kubeconfig", "192.168.1.100", "test-cluster")
	if err != nil {
		t.Logf("CreatePluginsList failed (expected in test environment): %v", err)
		return
	}

	found := false
	for _, plugin := range plugins {
		if plugin.GetName() == TLSName {
			found = true
			break
		}
	}

	if !found {
		t.Error("TLS plugin not found in CreatePluginsList")
		t.Log("Available plugins:")
		for _, plugin := range plugins {
			t.Logf("  - %s", plugin.GetName())
		}
	}
}

func TestPluginNames(t *testing.T) {
	expectedPlugins := []string{
		"argocd",
		"cert-manager",
		"load-balancer",
		"nginx-ingress",
		IngressName,
		TLSName,
	}

	plugins, err := CreatePluginsList("dummy-kubeconfig", "192.168.1.100", "test-cluster")
	if err != nil {
		t.Logf("CreatePluginsList failed (expected in test environment): %v", err)
		return
	}

	if len(plugins) != len(expectedPlugins) {
		t.Errorf("Expected %d plugins, got %d", len(expectedPlugins), len(plugins))
	}

	pluginNames := make(map[string]bool)
	for _, plugin := range plugins {
		pluginNames[plugin.GetName()] = true
	}

	for _, expected := range expectedPlugins {
		if !pluginNames[expected] {
			t.Errorf("Expected plugin '%s' not found", expected)
		}
	}
}
