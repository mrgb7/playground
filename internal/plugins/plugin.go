package plugins

type Plugin interface {
	GetName() string
	Install(kubeConfig, clusterName string, ensure ...bool) error
	Uninstall(kubeConfig, clusterName string, ensure ...bool) error
	Status() string
	GetOptions() PluginOptions
}

type PluginOptions struct {
	Version          *string
	Namespace        *string
	ChartName        *string
	RepoName         *string
	Repository       *string
	releaseName      *string
	ChartValues      map[string]interface{}
	CRDsGroupVersion string
}

func CreatePluginsList(kubeConfig, masterClusterIP, clusterName string) ([]Plugin, error) {
	lb, err := NewLoadBalancer(kubeConfig, masterClusterIP, clusterName)
	if err != nil {
		return nil, err
	}

	ingress, err := NewIngress(kubeConfig, clusterName)
	if err != nil {
		return nil, err
	}

	tls, err := NewTLS(kubeConfig, clusterName)
	if err != nil {
		return nil, err
	}
	argocd, err := NewArgocd(kubeConfig)
	if err != nil {
		return nil, err
	}

	return []Plugin{
		argocd,
		NewCertManager(kubeConfig),
		lb,
		NewNginx(kubeConfig),
		ingress,
		tls,
	}, nil
}
