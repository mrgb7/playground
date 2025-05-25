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
	expected := NginxNamespace
	if nginx.GetNamespace() != expected {
		t.Errorf("expected namespace %s, got %s", expected, nginx.GetNamespace())
	}
}

func TestNginx_GetVersion(t *testing.T) {
	nginx := NewNginx("")
	expected := NginxChartVersion
	if nginx.GetVersion() != expected {
		t.Errorf("expected version %s, got %s", expected, nginx.GetVersion())
	}
}

func TestNginx_GetChartName(t *testing.T) {
	nginx := NewNginx("")
	expected := "ingress-nginx"
	if nginx.GetChartName() != expected {
		t.Errorf("expected chart name %s, got %s", expected, nginx.GetChartName())
	}
}

func TestNginx_GetRepository(t *testing.T) {
	nginx := NewNginx("")
	expected := "https://kubernetes.github.io/ingress-nginx"
	if nginx.GetRepository() != expected {
		t.Errorf("expected repository %s, got %s", expected, nginx.GetRepository())
	}
}

func TestNginx_GetRepoName(t *testing.T) {
	nginx := NewNginx("")
	expected := "ingress-nginx"
	if nginx.GetRepoName() != expected {
		t.Errorf("expected repo name %s, got %s", expected, nginx.GetRepoName())
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

	if NginxNamespace != "ingress-nginx" {
		t.Errorf("expected NginxNamespace to be 'ingress-nginx', got '%s'", NginxNamespace)
	}

	if NginxChartVersion != "4.11.3" {
		t.Errorf("expected NginxChartVersion to be '4.11.3', got '%s'", NginxChartVersion)
	}
}

func TestNginx_Status_InvalidKubeConfig(t *testing.T) {
	nginx := NewNginx("invalid-config")
	status := nginx.Status()
	expected := "UNKNOWN"
	if status != expected {
		t.Errorf("expected status %s for invalid config, got %s", expected, status)
	}
}

func TestNginx_Status_EmptyKubeConfig(t *testing.T) {
	nginx := NewNginx("")
	status := nginx.Status()
	expected := "UNKNOWN"
	if status != expected {
		t.Errorf("expected status %s for empty config, got %s", expected, status)
	}
}
