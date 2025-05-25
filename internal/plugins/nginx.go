package plugins

import (
	"context"
	"time"

	"github.com/mrgb7/playground/internal/k8s"
	"github.com/mrgb7/playground/pkg/logger"
)

const (
	DefaultNginxReplicas = 2
	NginxNamespace       = "ingress-nginx"
	NginxChartVersion    = "4.11.3"
)

// Nginx represents the nginx-ingress controller plugin
// It provides LoadBalancer support for ingress traffic routing
type Nginx struct {
	KubeConfig string
	*BasePlugin
}

// NewNginx creates a new Nginx plugin instance with the provided kubeConfig
func NewNginx(kubeConfig string) *Nginx {
	nginx := &Nginx{
		KubeConfig: kubeConfig,
	}
	nginx.BasePlugin = NewBasePlugin(kubeConfig, nginx)
	return nginx
}

// GetName returns the plugin name used for identification
func (n *Nginx) GetName() string {
	return "nginx-ingress"
}

// Install deploys nginx-ingress controller using unified installation
func (n *Nginx) Install(kubeConfig, clusterName string, ensure ...bool) error {
	return n.UnifiedInstall(kubeConfig, clusterName, ensure...)
}

// Uninstall removes nginx-ingress controller using unified uninstallation
func (n *Nginx) Uninstall(kubeConfig, clusterName string, ensure ...bool) error {
	return n.UnifiedUninstall(kubeConfig, clusterName, ensure...)
}

// Status checks the current installation status of nginx-ingress
func (n *Nginx) Status() string {
	if n.KubeConfig == "" {
		logger.Errorln("kubeConfig is empty")
		return "UNKNOWN"
	}

	c, err := k8s.NewK8sClient(n.KubeConfig)
	if err != nil {
		logger.Errorln("failed to create k8s client: %v", err)
		return "UNKNOWN"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ns, err := c.GetNameSpace(NginxNamespace, ctx)
	if ns == "" || err != nil {
		logger.Debugln("nginx namespace not found or error occurred: %v", err)
		return StatusNotInstalled
	}
	return "nginx-ingress is running"
}

// GetNamespace returns the namespace where nginx-ingress is deployed
func (n *Nginx) GetNamespace() string {
	return NginxNamespace
}

// GetVersion returns the chart version to be installed
func (n *Nginx) GetVersion() string {
	return NginxChartVersion
}

// GetChartName returns the Helm chart name
func (n *Nginx) GetChartName() string {
	return "ingress-nginx"
}

// GetRepository returns the Helm repository URL
func (n *Nginx) GetRepository() string {
	return "https://kubernetes.github.io/ingress-nginx"
}

// GetRepoName returns the repository name for Helm
func (n *Nginx) GetRepoName() string {
	return "ingress-nginx"
}

// GetChartValues returns the configuration values for nginx-ingress
// Configures LoadBalancer service, metrics, security headers, and default backend
func (n *Nginx) GetChartValues() map[string]interface{} {
	return map[string]interface{}{
		"controller": map[string]interface{}{
			"replicaCount": DefaultNginxReplicas,
			"service": map[string]interface{}{
				"type": "LoadBalancer", // Generic LoadBalancer for any cloud provider
			},
			"config": map[string]interface{}{
				"enable-vts-status":          "true",  // Enable VTS status for monitoring
				"use-forwarded-headers":      "true",  // Handle forwarded headers properly
				"compute-full-forwarded-for": "true",  // Compute full forwarded-for chain
				"use-proxy-protocol":         "false", // Disable proxy protocol by default
			},
			"metrics": map[string]interface{}{
				"enabled": true, // Enable metrics collection
				"serviceMonitor": map[string]interface{}{
					"enabled": false, // Disable ServiceMonitor (can be enabled if Prometheus operator is available)
				},
			},
		},
		"defaultBackend": map[string]interface{}{
			"enabled": true, // Enable default backend for unmatched routes
		},
	}
}
