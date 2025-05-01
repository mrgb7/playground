package helm

import (
	"context"

	helmclient "github.com/mittwald/go-helm-client"
	"helm.sh/helm/v3/pkg/repo"
)

type Client struct {
	kubeconfig string
	namespace  string
	Wait       bool
	RepoName   string
	RepoURL    string
}

type InstallOptions struct {
	ReleaseName string
	ChartName   string
	Values      string
	RepoName    string
	RepoURL     string
}

func NewClient(kubeconfig, namespace string, wait bool) *Client {
	return &Client{
		kubeconfig: kubeconfig,
		namespace:  namespace,
		Wait:       wait,
	}
}

func (c *Client) Install(opt InstallOptions) error {
	c.RepoName = opt.RepoName
	c.RepoURL = opt.RepoURL
	hc, err := c.getHelmClient()
	if err != nil {
		return err
	}

	err = c.ensureRepo()
	if err != nil {
		return err
	}
	crSpec := &helmclient.ChartSpec{
		ReleaseName:     opt.ReleaseName,
		ChartName:       opt.ChartName,
		Namespace:       c.namespace,
		CreateNamespace: true,
	}
	if c.Wait {
		crSpec.Wait = true
	}
	_, err = hc.InstallOrUpgradeChart(context.Background(), crSpec, &helmclient.GenericHelmOptions{})

	return err
}

func (c *Client) Uninstall(releaseName string) error {
	hc, err := c.getHelmClient()
	if err != nil {
		return err
	}
	err = hc.UninstallReleaseByName(releaseName)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) getHelmClient() (helmclient.Client, error) {
	opt := &helmclient.KubeConfClientOptions{
		Options: &helmclient.Options{
			Namespace: c.namespace,
		},
		KubeConfig: []byte(c.kubeconfig),
	}
	cl, err := helmclient.NewClientFromKubeConf(opt)
	if err != nil {
		return nil, err
	}
	return cl, nil
}

func (c *Client) ensureRepo() error {
	hc, err := c.getHelmClient()
	if err != nil {
		return err
	}
	err = hc.UpdateChartRepos()
	if err != nil {
		return err
	}
	hc.AddOrUpdateChartRepo(repo.Entry{
		Name: c.RepoName,
		URL:  c.RepoURL,
	})
	return nil
}
