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
	return n.UnifiedInstall(kubeConfig, clusterName, ensure...)
}

func (n *Nginx) Uninstall(kubeConfig, clusterName string, ensure ...bool) error {
	return n.UnifiedUninstall(kubeConfig, clusterName, ensure...)
}

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

func (n *Nginx) GetNamespace() string {
	return NginxNamespace
}

func (n *Nginx) GetVersion() string {
	return NginxChartVersion
}

func (n *Nginx) GetChartName() string {
	return "ingress-nginx"
}

func (n *Nginx) GetRepository() string {
	return "https://kubernetes.github.io/ingress-nginx"
}

func (n *Nginx) GetRepoName() string {
	return "ingress-nginx"
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