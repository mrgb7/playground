package plugins

import (
	"reflect"
	"testing"
)

// MockDependencyPlugin for testing
type MockDependencyPlugin struct {
	name         string
	dependencies []string
}

func (m *MockDependencyPlugin) GetName() string           { return m.name }
func (m *MockDependencyPlugin) GetDependencies() []string { return m.dependencies }
func (m *MockDependencyPlugin) Install(kubeConfig, clusterName string, ensure ...bool) error {
	return nil
}
func (m *MockDependencyPlugin) Uninstall(kubeConfig, clusterName string, ensure ...bool) error {
	return nil
}
func (m *MockDependencyPlugin) Status() string                         { return "mock" }
func (m *MockDependencyPlugin) GetNamespace() string                   { return "test" }
func (m *MockDependencyPlugin) GetVersion() string                     { return "1.0.0" }
func (m *MockDependencyPlugin) GetChartName() string                   { return "test" }
func (m *MockDependencyPlugin) GetRepository() string                  { return "test" }
func (m *MockDependencyPlugin) GetRepoName() string                    { return "test" }
func (m *MockDependencyPlugin) GetChartValues() map[string]interface{} { return nil }
func (m *MockDependencyPlugin) GetOptions() PluginOptions {
	version := "1.0.0"
	namespace := "test"
	chartName := "test"
	repoName := "test"
	repository := "test"
	return PluginOptions{
		Version:     &version,
		Namespace:   &namespace,
		ChartName:   &chartName,
		RepoName:    &repoName,
		Repository:  &repository,
		ChartValues: nil,
	}
}

func TestDependencyGraph_AddPlugin(t *testing.T) {
	graph := NewDependencyGraph()

	plugin := &MockDependencyPlugin{
		name:         "test-plugin",
		dependencies: []string{"dep1", "dep2"},
	}

	graph.AddPlugin(plugin)

	if len(graph.nodes) != 3 { // test-plugin + dep1 + dep2 nodes
		t.Errorf("Expected 3 nodes, got %d", len(graph.nodes))
	}

	node := graph.nodes["test-plugin"]
	if node == nil {
		t.Fatal("test-plugin node not found")
	}

	if !reflect.DeepEqual(node.Dependencies, []string{"dep1", "dep2"}) {
		t.Errorf("Dependencies not stored correctly, got %v", node.Dependencies)
	}

	// Check that dependency nodes have this plugin as dependent
	dep1Node := graph.nodes["dep1"]
	if dep1Node == nil {
		t.Fatal("dep1 node not found")
	}
	if len(dep1Node.Dependents) != 1 || dep1Node.Dependents[0] != "test-plugin" {
		t.Errorf("Expected dep1 to have test-plugin as dependent, got %v", dep1Node.Dependents)
	}
}

func TestDependencyGraph_GetInstallOrder(t *testing.T) {
	graph := NewDependencyGraph()

	// Create plugins with dependencies: A -> B -> C
	pluginA := &MockDependencyPlugin{name: "A", dependencies: []string{"B"}}
	pluginB := &MockDependencyPlugin{name: "B", dependencies: []string{"C"}}
	pluginC := &MockDependencyPlugin{name: "C", dependencies: []string{}}

	graph.AddPlugin(pluginA)
	graph.AddPlugin(pluginB)
	graph.AddPlugin(pluginC)

	order, err := graph.GetInstallOrder([]string{"A"})
	if err != nil {
		t.Fatalf("GetInstallOrder failed: %v", err)
	}

	expected := []string{"C", "B", "A"}
	if !reflect.DeepEqual(order, expected) {
		t.Errorf("Expected install order %v, got %v", expected, order)
	}
}

func TestDependencyGraph_GetUninstallOrder(t *testing.T) {
	graph := NewDependencyGraph()

	// Create plugins with dependencies: A -> B -> C
	pluginA := &MockDependencyPlugin{name: "A", dependencies: []string{"B"}}
	pluginB := &MockDependencyPlugin{name: "B", dependencies: []string{"C"}}
	pluginC := &MockDependencyPlugin{name: "C", dependencies: []string{}}

	graph.AddPlugin(pluginA)
	graph.AddPlugin(pluginB)
	graph.AddPlugin(pluginC)

	order, err := graph.GetUninstallOrder([]string{"C"})
	if err != nil {
		t.Fatalf("GetUninstallOrder failed: %v", err)
	}

	expected := []string{"A", "B", "C"}
	if !reflect.DeepEqual(order, expected) {
		t.Errorf("Expected uninstall order %v, got %v", expected, order)
	}
}

func TestDependencyGraph_ValidateInstall(t *testing.T) {
	graph := NewDependencyGraph()

	plugin := &MockDependencyPlugin{
		name:         "test-plugin",
		dependencies: []string{"dep1", "dep2"},
	}
	graph.AddPlugin(plugin)

	// Test with missing dependencies
	err := graph.ValidateInstall("test-plugin", []string{})
	if err == nil {
		t.Error("Expected error for missing dependencies")
	}

	// Test with partial dependencies
	err = graph.ValidateInstall("test-plugin", []string{"dep1"})
	if err == nil {
		t.Error("Expected error for partial dependencies")
	}

	// Test with all dependencies
	err = graph.ValidateInstall("test-plugin", []string{"dep1", "dep2"})
	if err != nil {
		t.Errorf("Unexpected error with all dependencies: %v", err)
	}
}

func TestDependencyGraph_ValidateUninstall(t *testing.T) {
	graph := NewDependencyGraph()

	pluginA := &MockDependencyPlugin{name: "A", dependencies: []string{"B"}}
	pluginB := &MockDependencyPlugin{name: "B", dependencies: []string{}}

	graph.AddPlugin(pluginA)
	graph.AddPlugin(pluginB)

	// Test uninstalling B when A depends on it
	err := graph.ValidateUninstall("B", []string{"A", "B"})
	if err == nil {
		t.Error("Expected error when trying to uninstall plugin with dependents")
	}

	// Test uninstalling A (no dependents)
	err = graph.ValidateUninstall("A", []string{"A", "B"})
	if err != nil {
		t.Errorf("Unexpected error uninstalling plugin without dependents: %v", err)
	}
}

func TestDependencyGraph_HasCycles(t *testing.T) {
	graph := NewDependencyGraph()

	// Create circular dependency: A -> B -> C -> A
	pluginA := &MockDependencyPlugin{name: "A", dependencies: []string{"B"}}
	pluginB := &MockDependencyPlugin{name: "B", dependencies: []string{"C"}}
	pluginC := &MockDependencyPlugin{name: "C", dependencies: []string{"A"}}

	graph.AddPlugin(pluginA)
	graph.AddPlugin(pluginB)
	graph.AddPlugin(pluginC)

	if !graph.HasCycles() {
		t.Error("Expected cycle detection to return true")
	}

	// Test without cycles
	graph2 := NewDependencyGraph()
	pluginD := &MockDependencyPlugin{name: "D", dependencies: []string{"E"}}
	pluginE := &MockDependencyPlugin{name: "E", dependencies: []string{}}

	graph2.AddPlugin(pluginD)
	graph2.AddPlugin(pluginE)

	if graph2.HasCycles() {
		t.Error("Expected cycle detection to return false")
	}
}

func TestDependencyValidator_ValidateInstallation(t *testing.T) {
	plugins := []DependencyPlugin{
		&MockDependencyPlugin{name: "A", dependencies: []string{"B"}},
		&MockDependencyPlugin{name: "B", dependencies: []string{"C"}},
		&MockDependencyPlugin{name: "C", dependencies: []string{}},
	}

	validator := NewDependencyValidator(plugins)

	order, err := validator.ValidateInstallation([]string{"A"}, []string{})
	if err != nil {
		t.Fatalf("ValidateInstallation failed: %v", err)
	}

	expected := []string{"C", "B", "A"}
	if !reflect.DeepEqual(order, expected) {
		t.Errorf("Expected install order %v, got %v", expected, order)
	}
}

func TestDependencyValidator_ValidateUninstallation(t *testing.T) {
	plugins := []DependencyPlugin{
		&MockDependencyPlugin{name: "A", dependencies: []string{"B"}},
		&MockDependencyPlugin{name: "B", dependencies: []string{"C"}},
		&MockDependencyPlugin{name: "C", dependencies: []string{}},
	}

	validator := NewDependencyValidator(plugins)

	order, err := validator.ValidateUninstallation([]string{"C"}, []string{"A", "B", "C"})
	if err != nil {
		t.Fatalf("ValidateUninstallation failed: %v", err)
	}

	expected := []string{"A", "B", "C"}
	if !reflect.DeepEqual(order, expected) {
		t.Errorf("Expected uninstall order %v, got %v", expected, order)
	}
}

func TestDependencyValidator_GetDependencyInfo(t *testing.T) {
	plugins := []DependencyPlugin{
		&MockDependencyPlugin{name: "A", dependencies: []string{"B"}},
		&MockDependencyPlugin{name: "B", dependencies: []string{"C"}},
		&MockDependencyPlugin{name: "C", dependencies: []string{}},
	}

	validator := NewDependencyValidator(plugins)

	deps, dependents := validator.GetDependencyInfo("B")

	expectedDeps := []string{"C"}
	expectedDependents := []string{"A"}

	if !reflect.DeepEqual(deps, expectedDeps) {
		t.Errorf("Expected dependencies %v, got %v", expectedDeps, deps)
	}

	if !reflect.DeepEqual(dependents, expectedDependents) {
		t.Errorf("Expected dependents %v, got %v", expectedDependents, dependents)
	}
}

func TestRealPluginDependencies(t *testing.T) {
	// Test real plugin dependencies
	argocd, err := NewArgocd("")
	if err != nil {
		t.Skipf("Skipping ArgoCD test due to initialization error: %v", err)
	}
	certManager := NewCertManager("")
	nginx := NewNginx("")

	// Test that plugins implement DependencyPlugin interface
	if argocd != nil {
		var _ DependencyPlugin = argocd
	}
	var _ DependencyPlugin = certManager
	var _ DependencyPlugin = nginx

	// Test dependency declarations
	if argocd != nil && len(argocd.GetDependencies()) != 0 {
		t.Errorf("ArgoCD should have no dependencies, got %v", argocd.GetDependencies())
	}

	if len(certManager.GetDependencies()) != 0 {
		t.Errorf("CertManager should have no dependencies, got %v", certManager.GetDependencies())
	}

	nginxDeps := nginx.GetDependencies()
	expectedNginxDeps := []string{"load-balancer"}
	if !reflect.DeepEqual(nginxDeps, expectedNginxDeps) {
		t.Errorf("Nginx dependencies should be %v, got %v", expectedNginxDeps, nginxDeps)
	}
}

func TestComplexDependencyScenario(t *testing.T) {
	// Test a complex scenario similar to real plugin dependencies
	plugins := []DependencyPlugin{
		&MockDependencyPlugin{name: "argocd", dependencies: []string{}},
		&MockDependencyPlugin{name: "cert-manager", dependencies: []string{}},
		&MockDependencyPlugin{name: "load-balancer", dependencies: []string{}},
		&MockDependencyPlugin{name: "nginx-ingress", dependencies: []string{"load-balancer"}},
		&MockDependencyPlugin{name: "ingress", dependencies: []string{"nginx-ingress", "load-balancer"}},
		&MockDependencyPlugin{name: "tls", dependencies: []string{"cert-manager"}},
	}

	validator := NewDependencyValidator(plugins)

	// Test installing ingress (should install load-balancer and nginx-ingress first)
	order, err := validator.ValidateInstallation([]string{"ingress"}, []string{})
	if err != nil {
		t.Fatalf("Failed to validate ingress installation: %v", err)
	}

	// Check that load-balancer comes before nginx-ingress and ingress
	lbIndex := indexOf(order, "load-balancer")
	nginxIndex := indexOf(order, "nginx-ingress")
	ingressIndex := indexOf(order, "ingress")

	if lbIndex == -1 || nginxIndex == -1 || ingressIndex == -1 {
		t.Fatalf("Missing plugins in install order: %v", order)
	}

	if lbIndex >= nginxIndex || nginxIndex >= ingressIndex {
		t.Errorf("Invalid install order: %v", order)
	}

	// Test uninstalling load-balancer when nginx-ingress and ingress depend on it
	_, err = validator.ValidateUninstallation([]string{"load-balancer"}, []string{"load-balancer", "nginx-ingress", "ingress"})
	if err != nil {
		t.Fatalf("Failed to validate load-balancer uninstallation: %v", err)
	}
}

func indexOf(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}

func TestMultiplePluginInstallation(t *testing.T) {
	plugins := []DependencyPlugin{
		&MockDependencyPlugin{name: "A", dependencies: []string{}},
		&MockDependencyPlugin{name: "B", dependencies: []string{"A"}},
		&MockDependencyPlugin{name: "C", dependencies: []string{"A"}},
		&MockDependencyPlugin{name: "D", dependencies: []string{"B", "C"}},
	}

	validator := NewDependencyValidator(plugins)

	// Install multiple plugins at once
	order, err := validator.ValidateInstallation([]string{"B", "C", "D"}, []string{})
	if err != nil {
		t.Fatalf("Failed to validate multiple plugin installation: %v", err)
	}

	// A should be first, D should be last
	aIndex := indexOf(order, "A")
	dIndex := indexOf(order, "D")

	if aIndex != 0 {
		t.Errorf("A should be first in install order, got index %d", aIndex)
	}

	if dIndex != len(order)-1 {
		t.Errorf("D should be last in install order, got index %d", dIndex)
	}
}
