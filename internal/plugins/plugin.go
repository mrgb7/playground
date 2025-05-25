package plugins

type Plugin interface {
	GetName() string
	Install(kubeConfig, clusterName string, ensure ...bool) error
	Uninstall(kubeConfig, clusterName string, ensure ...bool) error
	Status() string
	GetNamespace() string
	GetVersion() string
	GetChartName() string
	GetRepository() string
	GetRepoName() string
	GetChartValues() map[string]interface{}
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
		NewNginx(kubeConfig),
	}, nil
}

func GetBasicPluginsList() []Plugin {
	return []Plugin{
		NewArgocd(""),
		NewCertManager(""),
		NewNginx(""),
	}
}

var List = GetBasicPluginsList()
