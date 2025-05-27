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

const (
	CertManagerRepoURL      = "https://charts.jetstack.io"
	CertManagerChartName    = "cert-manager"
	CertManagerChartVersion = "v1.17.2"
	CertManagerReleaseName  = "cert-manager"
	CertManagerNamespace    = "cert-manager"
	CertManagerRepoName     = "jetstack"

	DefaultWebhookTimeout = 10
)

func NewCertManager(kubeConfig string) *CertManager {
	cm := &CertManager{
		KubeConfig: kubeConfig,
	}
	cm.BasePlugin = NewBasePlugin(kubeConfig, cm)
	return cm
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
		logger.Errorln("failed to create k8s client: %v", err)
		return StatusUnknown
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ns, err := client.GetNameSpace(CertManagerNamespace, ctx)
	if ns == "" || err != nil {
		logger.Errorln("failed to get cert-manager namespace: %v", err)
		return StatusNotInstalled
	}
	return "cert-manager is running"
}

func (c *CertManager) GetNamespace() string {
	return CertManagerNamespace
}

func (c *CertManager) GetVersion() string {
	return CertManagerChartVersion
}

func (c *CertManager) GetChartName() string {
	return CertManagerChartName
}

func (c *CertManager) GetRepository() string {
	return CertManagerRepoURL
}

func (c *CertManager) GetChartValues() map[string]interface{} {
	return c.getDefaultValues()
}

func (c *CertManager) GetReleaseName() string {
	return CertManagerReleaseName
}

func (c *CertManager) GetRepoName() string {
	return CertManagerRepoName
}
