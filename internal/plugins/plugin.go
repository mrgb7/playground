package plugins

import "github.com/mrgb7/playground/internal/installer"

type Plugin interface {
	GetName() string
	GetInstaller() (installer.Installer, error)
	Install(ensure ...bool) error
	Uninstall(ensure ...bool) error
	Status() string
}

var List = []Plugin{
	&Argocd{},
	&CertManager{},
	&LoadBalancer{},
	&Nginx{},
}
