package plugins

import "github.com/mrgb7/playground/internal/installer"

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
