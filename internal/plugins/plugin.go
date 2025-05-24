package plugins

import "github.com/mrgb7/playground/internal/installer"

type Plugin interface {
	GetName() string
	GetInstaller() (installer.Installer, error)
	Install(kubeConfig, clusterName string, ensure ...bool) error
	Uninstall(kubeConfig, clusterName string, ensure ...bool) error
	Status() string
}

func CreatePluginsList(kubeConfig, masterClusterIP string) ([]Plugin, error) {
	lb, err := NewLoadBalancer(kubeConfig, masterClusterIP)
	if err != nil {
		return nil, err
	}
	
	return []Plugin{
		NewArgocd(kubeConfig),
		NewCertManager(kubeConfig),
		lb,
		&Nginx{},
	}, nil
}

var List = []Plugin{
	&Argocd{},
	&CertManager{},
	&LoadBalancer{},
	&Nginx{},
}
