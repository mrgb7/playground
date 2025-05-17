package plugins

import "github.com/mohamedragab2024/playground/internal/installer"

type Nginx struct{}

func (n *Nginx) GetName() string {
	return "nginx"
}

func (n *Nginx) GetInstaller() (installer.Installer, error) {
	return nil, nil
}

func (n *Nginx) Install(ensure ...bool) error {
	return nil
}

func (n *Nginx) Uninstall(ensure ...bool) error {
	return nil
}

func (n *Nginx) Status() string {
	return "nginx is running"
}
