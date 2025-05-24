package plugins

import (
	"context"
	"time"

	"github.com/mrgb7/playground/internal/installer"
	"github.com/mrgb7/playground/internal/k8s"
	"github.com/mrgb7/playground/pkg/logger"
)

type CertManager struct {
	KubeConfig string
	*BasePlugin
}

const (
	CertManagerRepoUrl       = "https://charts.jetstack.io"
	CertManagerChartName     = "cert-manager"
	CertManagerChartVersion  = "v1.17.2"
	CertManagerReleaseName   = "cert-manager"
	CertManagerNamespace     = "cert-manager"
	CertManagerRepoName      = "jetstack"
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
	return c.BasePlugin.UnifiedInstall(kubeConfig, clusterName, ensure...)
}

func (c *CertManager) Uninstall(kubeConfig, clusterName string, ensure ...bool) error {
	return c.BasePlugin.UnifiedUninstall(kubeConfig, clusterName, ensure...)
}

func (c *CertManager) GetInstaller() (installer.Installer, error) {
	values := c.getDefaultValues()
	return &installer.HelmInstaller{
		ReleaseName:  CertManagerReleaseName,
		ChartName:    CertManagerChartName,
		RepoUrl:      CertManagerRepoUrl,
		RepoName:     CertManagerRepoName,
		Namespace:    CertManagerNamespace,
		ChartVersion: CertManagerChartVersion,
		Values:       values,
		KubeConfig:   c.KubeConfig,
	}, nil
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
			"timeoutSeconds": 10,
		},
	}
}

func (c *CertManager) Status() string {
	client, err := k8s.NewK8sClient(c.KubeConfig)
	if err != nil {
		logger.Errorln("failed to create k8s client: %v", err)
		return "UNKNOWN"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	ns, err := client.GetNameSpace(CertManagerNamespace, ctx)
	if ns == "" || err != nil {
		logger.Errorln("failed to get cert-manager namespace: %v", err)
		return "Not installed"
	}
	return "cert-manager is running"
}
