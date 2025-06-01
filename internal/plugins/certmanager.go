package plugins

import (
	"context"
	"time"

	"github.com/mrgb7/playground/internal/k8s"
	"github.com/mrgb7/playground/pkg/logger"
)

type CertManager struct {
	KubeConfig string
	*BasePlugin
}

var (
	CertManagerRepoURL      = "https://charts.jetstack.io"
	CertManagerChartName    = "cert-manager"
	CertManagerChartVersion = "v1.17.2"
	CertManagerReleaseName  = "cert-manager"
	CertManagerNamespace    = "cert-manager"
	CertManagerRepoName     = "jetstack"
)

const (
	DefaultWebhookTimeout = 10
)

func NewCertManager(kubeConfig string) *CertManager {
	cm := &CertManager{
		KubeConfig: kubeConfig,
	}
	cm.BasePlugin = NewBasePlugin(kubeConfig, cm)
	return cm
}

func (c *CertManager) GetOptions() PluginOptions {
	return PluginOptions{
		Version:          &CertManagerChartVersion,
		Namespace:        &CertManagerNamespace,
		ChartName:        &CertManagerChartName,
		RepoName:         &CertManagerRepoName,
		Repository:       &CertManagerRepoURL,
		ChartValues:      c.getDefaultValues(),
		CRDsGroupVersion: "cert-manager.io",
	}
}

func (c *CertManager) GetName() string {
	return "cert-manager"
}

func (c *CertManager) Install(kubeConfig, clusterName string, ensure ...bool) error {
	return c.UnifiedInstall(kubeConfig, clusterName, ensure...)
}

func (c *CertManager) Uninstall(kubeConfig, clusterName string, ensure ...bool) error {
	return c.UnifiedUninstall(kubeConfig, clusterName, ensure...)
}

func (c *CertManager) getDefaultValues() map[string]interface{} {
	return map[string]interface{}{
		"crds": map[string]interface{}{
			"enabled": true,
		},
		"prometheus": map[string]interface{}{
			"enabled": true,
		},
		"webhook": map[string]interface{}{
			"timeoutSeconds": DefaultWebhookTimeout,
		},
	}
}

func (c *CertManager) Status() string {
	client, err := k8s.NewK8sClient(c.KubeConfig)
	if err != nil {
		logger.Debugf("failed to create k8s client: %v", err)
		return StatusUnknown
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ns, err := client.GetNameSpace(CertManagerNamespace, ctx)
	if ns == "" || err != nil {
		logger.Debugf("cert-manager namespace not found or error occurred: %v", err)
		return StatusNotInstalled
	}
	return StatusRunning
}

func (c *CertManager) GetDependencies() []string {
	return []string{} // cert-manager has no dependencies
}
