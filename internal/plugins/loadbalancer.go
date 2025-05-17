package plugins

import "github.com/mohamedragab2024/playground/internal/installer"

type LoadBalancer struct{}

func (l *LoadBalancer) GetName() string {
	return "LoadBalancer"
}

func (l *LoadBalancer) GetInstaller() (installer.Installer, error) {
	return nil, nil
}

func (l *LoadBalancer) Install(ensure ...bool) error {
	return nil
}

func (l *LoadBalancer) Uninstall(ensure ...bool) error {
	return nil
}

func (l *LoadBalancer) Status() string {
	return "LoadBalancer is running"
}
