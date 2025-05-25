package plugins

import "github.com/mrgb7/playground/internal/installer"

const (
	DefaultNginxReplicas = 2
)

type Nginx struct{}

func (n *Nginx) GetName() string {
	return "nginx"
}

func (n *Nginx) GetInstaller() (installer.Installer, error) {
	return nil, nil
}

func (n *Nginx) Install(kubeConfig, clusterName string, ensure ...bool) error {
	return nil
}

func (n *Nginx) Uninstall(kubeConfig, clusterName string, ensure ...bool) error {
	return nil
}

func (n *Nginx) Status() string {
	return "nginx is running"
}

func (n *Nginx) GetNamespace() string {
	return "nginx"
}

func (n *Nginx) GetVersion() string {
	return "1.21.6"
}

func (n *Nginx) GetChartName() string {
	return "nginx-ingress"
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
				"enable-vts-status": "true",
			},
		},
	}
}
