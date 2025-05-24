package installer

type Installer interface {
	Install(options *InstallOptions) error
	UnInstall(options *InstallOptions) error
}

type InstallOptions struct{}
