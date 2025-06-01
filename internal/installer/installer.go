package installer

type Installer interface {
	Install(options *InstallOptions) error
	UnInstall(options *InstallOptions) error
}

type InstallOptions struct {
	ApplicationName  string
	RepoURL          string
	Path             string
	Version          string
	Namespace        string
	ChartName        *string
	Values           map[string]interface{}
	KubeConfig       string
	RepoName         string
	CRDsGroupVersion string
}
