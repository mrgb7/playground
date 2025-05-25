package plugins

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mrgb7/playground/internal/k8s"
	"github.com/mrgb7/playground/pkg/logger"
)

const (
	DefaultNginxReplicas = 2
	NginxNamespace       = "ingress-nginx"
	NginxChartVersion    = "4.11.3"
	NginxChartName       = "ingress-nginx"
	NginxRepoName        = "ingress-nginx"
)

type Nginx struct {
	KubeConfig string
	*BasePlugin
}

func NewNginx(kubeConfig string) *Nginx {
	nginx := &Nginx{
		KubeConfig: kubeConfig,
	}
	nginx.BasePlugin = NewBasePlugin(kubeConfig, nginx)
	return nginx
}

func (n *Nginx) GetName() string {
	return "nginx-ingress"
}

func (n *Nginx) Install(kubeConfig, clusterName string, ensure ...bool) error {
	// Check dependencies before installation
	if err := n.checkDependencies(kubeConfig); err != nil {
		return fmt.Errorf("dependency check failed: %w", err)
	}
	
	return n.UnifiedInstall(kubeConfig, clusterName, ensure...)
}

func (n *Nginx) Uninstall(kubeConfig, clusterName string, ensure ...bool) error {
	return n.UnifiedUninstall(kubeConfig, clusterName, ensure...)
}

func (n *Nginx) Status() string {
	if n.KubeConfig == "" {
		logger.Errorf("kubeConfig is empty")
		return StatusUnknown
	}

	c, err := k8s.NewK8sClient(n.KubeConfig)
	if err != nil {
		logger.Errorf("failed to create k8s client: %v", err)
		return StatusUnknown
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ns, err := c.GetNameSpace(NginxNamespace, ctx)
	if ns == "" || err != nil {
		logger.Debugf("nginx namespace not found or error occurred: %v", err)
		return StatusNotInstalled
	}
	return n.GetName() + " is " + StatusRunning
}

func (n *Nginx) checkDependencies(kubeConfig string) error {
	logger.Infoln("Checking nginx-ingress dependencies...")

	// Check for load-balancer dependency
	lb, err := NewLoadBalancer(kubeConfig, "")
	if err != nil {
		return fmt.Errorf("failed to create loadbalancer client: %w", err)
	}
	
	lbStatus := lb.Status()
	if !strings.Contains(lbStatus, StatusRunning) {
		return fmt.Errorf("load-balancer plugin is required but not installed/running. Status: %s", lbStatus)
	}

	logger.Successln("All dependencies satisfied")
	return nil
}

func (n *Nginx) GetNamespace() string {
	return NginxNamespace
}

func (n *Nginx) GetVersion() string {
	return NginxChartVersion
}

func (n *Nginx) GetChartName() string {
	return NginxChartName
}

func (n *Nginx) GetRepository() string {
	return "https://kubernetes.github.io/ingress-nginx"
}

func (n *Nginx) GetRepoName() string {
	return NginxRepoName
}

func (n *Nginx) GetChartValues() map[string]interface{} {
	return map[string]interface{}{
		"controller": map[string]interface{}{
			"replicaCount": DefaultNginxReplicas,
			"service": map[string]interface{}{
				"type": "LoadBalancer",
			},
			"config": map[string]interface{}{
				"enable-vts-status":          "true",
				"use-forwarded-headers":      "true",
				"compute-full-forwarded-for": "true",
				"use-proxy-protocol":         "false",
			},
			"metrics": map[string]interface{}{
				"enabled": true,
				"serviceMonitor": map[string]interface{}{
					"enabled": false,
				},
			},
		},
		"defaultBackend": map[string]interface{}{
			"enabled": true,
		},
	}
}
