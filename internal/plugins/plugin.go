package plugins

import "github.com/mrgb7/playground/internal/installer"

type Plugin interface {
	GetName() string
	GetInstaller() (installer.Installer, error)
	Install(ensure ...bool) error
	Uninstall(ensure ...bool) error
	Status() string
}

type FactoryAwarePlugin interface {
	Plugin
	InstallWithFactory(kubeConfig, clusterName string, ensure ...bool) error
	UninstallWithFactory(kubeConfig, clusterName string, ensure ...bool) error
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
