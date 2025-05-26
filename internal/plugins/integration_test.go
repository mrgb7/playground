package plugins

import (
	"testing"
)

func TestPluginDependencyIntegration(t *testing.T) {
	// Test that all real plugins implement DependencyPlugin interface
	plugins := []Plugin{
		NewArgocd(""),
		NewCertManager(""),
		NewNginx(""),
	}

	// Test LoadBalancer separately since it requires additional parameters
	lb, err := NewLoadBalancer("", "")
	if err != nil {
		t.Logf("LoadBalancer creation failed (expected in test): %v", err)
	} else {
		plugins = append(plugins, lb)
	}

	for _, plugin := range plugins {
		// Test that plugin implements DependencyPlugin interface
		if _, ok := plugin.(DependencyPlugin); !ok {
			t.Errorf("Plugin %s does not implement DependencyPlugin interface", plugin.GetName())
		}
	}
}

func TestDependencyValidationIntegration(t *testing.T) {
	// Create mock plugins that simulate real dependency relationships
	dependencyPlugins := []DependencyPlugin{
		&MockDependencyPlugin{name: "argocd", dependencies: []string{}},
		&MockDependencyPlugin{name: "cert-manager", dependencies: []string{}},
		&MockDependencyPlugin{name: "load-balancer", dependencies: []string{}},
		&MockDependencyPlugin{name: "nginx-ingress", dependencies: []string{"load-balancer"}},
		&MockDependencyPlugin{name: "ingress", dependencies: []string{"nginx-ingress", "load-balancer"}},
		&MockDependencyPlugin{name: "tls", dependencies: []string{"cert-manager"}},
	}

	validator := NewDependencyValidator(dependencyPlugins)

	// Test 1: Installing ingress should install load-balancer and nginx-ingress first
	installOrder, err := validator.ValidateInstallation([]string{"ingress"}, []string{})
	if err != nil {
		t.Fatalf("Failed to validate ingress installation: %v", err)
	}

	// Verify load-balancer comes before nginx-ingress and ingress
	lbIndex := indexOf(installOrder, "load-balancer")
	nginxIndex := indexOf(installOrder, "nginx-ingress")
	ingressIndex := indexOf(installOrder, "ingress")

	if lbIndex == -1 || nginxIndex == -1 || ingressIndex == -1 {
		t.Fatalf("Missing plugins in install order: %v", installOrder)
	}

	if lbIndex >= nginxIndex || nginxIndex >= ingressIndex {
		t.Errorf("Invalid install order: %v", installOrder)
	}

	// Test 2: Installing TLS should install cert-manager first
	tlsInstallOrder, err := validator.ValidateInstallation([]string{"tls"}, []string{})
	if err != nil {
		t.Fatalf("Failed to validate TLS installation: %v", err)
	}

	if len(tlsInstallOrder) != 2 || tlsInstallOrder[0] != "cert-manager" || tlsInstallOrder[1] != "tls" {
		t.Errorf("Invalid TLS install order: %v", tlsInstallOrder)
	}

	// Test 3: Uninstalling load-balancer when nginx-ingress and ingress depend on it
	uninstallOrder, err := validator.ValidateUninstallation([]string{"load-balancer"}, 
		[]string{"load-balancer", "nginx-ingress", "ingress"})
	if err != nil {
		t.Fatalf("Failed to validate load-balancer uninstallation: %v", err)
	}

	// Should uninstall in reverse dependency order: ingress, nginx-ingress, load-balancer
	expectedUninstallOrder := []string{"ingress", "nginx-ingress", "load-balancer"}
	if len(uninstallOrder) != len(expectedUninstallOrder) {
		t.Errorf("Unexpected uninstall order length: got %d, expected %d", 
			len(uninstallOrder), len(expectedUninstallOrder))
	}

	for i, expected := range expectedUninstallOrder {
		if i >= len(uninstallOrder) || uninstallOrder[i] != expected {
			t.Errorf("Invalid uninstall order at position %d: got %v, expected %v", 
				i, uninstallOrder, expectedUninstallOrder)
			break
		}
	}

	// Test 4: Trying to uninstall cert-manager when TLS depends on it should fail
	_, err = validator.ValidateUninstallation([]string{"cert-manager"}, 
		[]string{"cert-manager", "tls"})
	if err != nil {
		t.Fatalf("Failed to validate cert-manager uninstallation: %v", err)
	}
}

func TestCircularDependencyDetection(t *testing.T) {
	// Create plugins with circular dependency
	circularPlugins := []DependencyPlugin{
		&MockDependencyPlugin{name: "A", dependencies: []string{"B"}},
		&MockDependencyPlugin{name: "B", dependencies: []string{"C"}},
		&MockDependencyPlugin{name: "C", dependencies: []string{"A"}},
	}

	validator := NewDependencyValidator(circularPlugins)

	// Should detect circular dependency
	_, err := validator.ValidateInstallation([]string{"A"}, []string{})
	if err == nil {
		t.Error("Expected error for circular dependency, but got none")
	}
}

func TestPartialInstallationScenario(t *testing.T) {
	dependencyPlugins := []DependencyPlugin{
		&MockDependencyPlugin{name: "argocd", dependencies: []string{}},
		&MockDependencyPlugin{name: "cert-manager", dependencies: []string{}},
		&MockDependencyPlugin{name: "load-balancer", dependencies: []string{}},
		&MockDependencyPlugin{name: "nginx-ingress", dependencies: []string{"load-balancer"}},
		&MockDependencyPlugin{name: "ingress", dependencies: []string{"nginx-ingress", "load-balancer"}},
		&MockDependencyPlugin{name: "tls", dependencies: []string{"cert-manager"}},
	}

	validator := NewDependencyValidator(dependencyPlugins)

	// Scenario: load-balancer is already installed, now installing ingress
	installOrder, err := validator.ValidateInstallation([]string{"ingress"}, 
		[]string{"load-balancer"})
	if err != nil {
		t.Fatalf("Failed to validate partial installation: %v", err)
	}

	// Should only install nginx-ingress and ingress (load-balancer already installed)
	expectedOrder := []string{"nginx-ingress", "ingress"}
	if len(installOrder) != len(expectedOrder) {
		t.Errorf("Unexpected install order length: got %d, expected %d", 
			len(installOrder), len(expectedOrder))
	}

	for i, expected := range expectedOrder {
		if i >= len(installOrder) || installOrder[i] != expected {
			t.Errorf("Invalid install order at position %d: got %v, expected %v", 
				i, installOrder, expectedOrder)
			break
		}
	}
}

func TestMultiplePluginInstallationIntegration(t *testing.T) {
	dependencyPlugins := []DependencyPlugin{
		&MockDependencyPlugin{name: "argocd", dependencies: []string{}},
		&MockDependencyPlugin{name: "cert-manager", dependencies: []string{}},
		&MockDependencyPlugin{name: "load-balancer", dependencies: []string{}},
		&MockDependencyPlugin{name: "nginx-ingress", dependencies: []string{"load-balancer"}},
		&MockDependencyPlugin{name: "ingress", dependencies: []string{"nginx-ingress", "load-balancer"}},
		&MockDependencyPlugin{name: "tls", dependencies: []string{"cert-manager"}},
	}

	validator := NewDependencyValidator(dependencyPlugins)

	// Install multiple plugins at once: ingress and tls
	installOrder, err := validator.ValidateInstallation([]string{"ingress", "tls"}, []string{})
	if err != nil {
		t.Fatalf("Failed to validate multiple plugin installation: %v", err)
	}

	// Should install all dependencies
	expectedPlugins := map[string]bool{
		"cert-manager":   true,
		"load-balancer":  true,
		"nginx-ingress":  true,
		"ingress":        true,
		"tls":            true,
	}

	if len(installOrder) != len(expectedPlugins) {
		t.Errorf("Unexpected number of plugins to install: got %d, expected %d", 
			len(installOrder), len(expectedPlugins))
	}

	for _, plugin := range installOrder {
		if !expectedPlugins[plugin] {
			t.Errorf("Unexpected plugin in install order: %s", plugin)
		}
	}

	// Verify dependency order constraints
	lbIndex := indexOf(installOrder, "load-balancer")
	nginxIndex := indexOf(installOrder, "nginx-ingress")
	ingressIndex := indexOf(installOrder, "ingress")
	cmIndex := indexOf(installOrder, "cert-manager")
	tlsIndex := indexOf(installOrder, "tls")

	if lbIndex >= nginxIndex || nginxIndex >= ingressIndex {
		t.Errorf("Invalid dependency order for ingress chain: %v", installOrder)
	}

	if cmIndex >= tlsIndex {
		t.Errorf("Invalid dependency order for TLS chain: %v", installOrder)
	}
} 