package plugins

import "github.com/mrgb7/playground/internal/installer"

type CertManager struct{}

func (c *CertManager) GetName() string {
	return "cert-manager"
}

func (c *CertManager) GetInstaller() (installer.Installer, error) {
	return nil, nil
}

func (c *CertManager) Install(ensure ...bool) error {
	return nil
}

func (c *CertManager) Uninstall(ensure ...bool) error {
	return nil
}

func (c *CertManager) Status() string {
	return "cert-manager is running"
}
