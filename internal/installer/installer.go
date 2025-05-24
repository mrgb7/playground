package installer

type Installer interface {
	Install(options *InstallOptions) error
	UnInstall(options *InstallOptions) error
}

type InstallOptions struct {
	ApplicationName string
	RepoURL         string
	Path            string
	TargetRevision  string
	Namespace       string
	Values          map[string]interface{}
}
