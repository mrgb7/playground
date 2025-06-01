package plugins

import (
	"testing"
)

func TestNewNginx(t *testing.T) {
	kubeConfig := "test-config"
	nginx := NewNginx(kubeConfig)

	if nginx.KubeConfig != kubeConfig {
		t.Errorf("expected KubeConfig %s, got %s", kubeConfig, nginx.KubeConfig)
	}

	if nginx.BasePlugin == nil {
		t.Errorf("BasePlugin should not be nil")
	}
}

func TestNginx_GetName(t *testing.T) {
	nginx := NewNginx("")
	expected := "nginx-ingress"
	if nginx.GetName() != expected {
		t.Errorf("expected name %s, got %s", expected, nginx.GetName())
	}
}

func TestNginx_GetNamespace(t *testing.T) {
	nginx := NewNginx("")
	options := nginx.GetOptions()
	expected := NginxNamespace
	if options.Namespace == nil || *options.Namespace != expected {
		t.Errorf("expected namespace %s, got %v", expected, options.Namespace)
	}
}

func TestNginx_GetVersion(t *testing.T) {
	nginx := NewNginx("")
	options := nginx.GetOptions()
	expected := NginxChartVersion
	if options.Version == nil || *options.Version != expected {
		t.Errorf("expected version %s, got %v", expected, options.Version)
	}
}

func TestNginx_GetChartName(t *testing.T) {
	nginx := NewNginx("")
	options := nginx.GetOptions()
	expected := NginxChartName
	if options.ChartName == nil || *options.ChartName != expected {
		t.Errorf("expected chart name %s, got %v", expected, options.ChartName)
	}
}

func TestNginx_GetRepository(t *testing.T) {
	nginx := NewNginx("")
	options := nginx.GetOptions()
	expected := "https://kubernetes.github.io/ingress-nginx"
	if options.Repository == nil || *options.Repository != expected {
		t.Errorf("expected repository %s, got %v", expected, options.Repository)
	}
}

func TestNginx_GetRepoName(t *testing.T) {
	nginx := NewNginx("")
	options := nginx.GetOptions()
	expected := NginxRepoName
	if options.RepoName == nil || *options.RepoName != expected {
		t.Errorf("expected repo name %s, got %v", expected, options.RepoName)
	}
}

func TestNginx_GetChartValues(t *testing.T) {
	nginx := NewNginx("")
	values := nginx.GetChartValues()

	if values == nil {
		t.Fatalf("GetChartValues should not return nil")
	}

	// Check that controller configuration exists
	controller, ok := values["controller"].(map[string]interface{})
	if !ok {
		t.Fatalf("controller configuration should exist")
	}

	// Check replica count
	replicaCount, ok := controller["replicaCount"].(int)
	if !ok || replicaCount != DefaultNginxReplicas {
		t.Errorf("expected replicaCount %d, got %v", DefaultNginxReplicas, replicaCount)
	}

	// Check service configuration
	service, ok := controller["service"].(map[string]interface{})
	if !ok {
		t.Fatalf("service configuration should exist")
	}

	serviceType, ok := service["type"].(string)
	if !ok || serviceType != "LoadBalancer" {
		t.Errorf("expected service type LoadBalancer, got %v", serviceType)
	}

	// Check that default backend is enabled
	defaultBackend, ok := values["defaultBackend"].(map[string]interface{})
	if !ok {
		t.Fatalf("defaultBackend configuration should exist")
	}

	enabled, ok := defaultBackend["enabled"].(bool)
	if !ok || !enabled {
		t.Errorf("expected defaultBackend to be enabled, got %v", enabled)
	}
}

func TestNginx_Constants(t *testing.T) {
	if DefaultNginxReplicas != 2 {
		t.Errorf("expected DefaultNginxReplicas to be 2, got %d", DefaultNginxReplicas)
	}

	if NginxNamespace != NginxChartName {
		t.Errorf("expected NginxNamespace to be '%s', got '%s'", NginxChartName, NginxNamespace)
	}

	if NginxChartVersion != "4.11.3" {
		t.Errorf("expected NginxChartVersion to be '4.11.3', got '%s'", NginxChartVersion)
	}
}

func TestNginx_Status_InvalidKubeConfig(t *testing.T) {
	nginx := NewNginx("invalid-config")
	status := nginx.Status()
	expected := StatusUnknown
	if status != expected {
		t.Errorf("expected status %s for invalid config, got %s", expected, status)
	}
}

func TestNginx_Status_EmptyKubeConfig(t *testing.T) {
	nginx := NewNginx("")
	status := nginx.Status()
	expected := StatusUnknown
	if status != expected {
		t.Errorf("expected status %s for empty config, got %s", expected, status)
	}
}

func TestNginx_Status_ValidConfig_NamespaceNotFound(t *testing.T) {
	nginx := NewNginx("~/.kube/config")
	status := nginx.Status()
	expected := StatusUnknown
	if status != expected {
		t.Errorf("expected status %s when kubeconfig is invalid, got %s", expected, status)
	}
}

func TestNginx_Status_Constants(t *testing.T) {
	if StatusRunning != "running" {
		t.Errorf("expected StatusRunning to be 'running', got '%s'", StatusRunning)
	}

	if StatusNotInstalled != "Not installed" {
		t.Errorf("expected StatusNotInstalled to be 'Not installed', got '%s'", StatusNotInstalled)
	}

	if StatusUnknown != "UNKNOWN" {
		t.Errorf("expected StatusUnknown to be 'UNKNOWN', got '%s'", StatusUnknown)
	}
}

func TestNginx_Status_Message_Format(t *testing.T) {
	nginx := NewNginx("valid-config")
	expectedFormat := nginx.GetName() + " is " + StatusRunning
	if expectedFormat != "nginx-ingress is running" {
		t.Errorf("expected status format 'nginx-ingress is running', got '%s'", expectedFormat)
	}
}
