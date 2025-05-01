package plugins

type (
	Plugin interface {
		GetName() string
		Install(kubeConfig string, ensure ...bool) error
		Uninstall(KubeConfig string, ensure ...bool) error
		GetValuesContent() (string, error)
	}
)
