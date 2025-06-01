package plugins

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/mrgb7/playground/internal/k8s"
	"github.com/mrgb7/playground/pkg/logger"
	"gopkg.in/yaml.v3"
)

type Argocd struct {
	KubeConfig string
	*BasePlugin
	Tracker *InstallerTracker
}

var (
	ArgocdRepoURL       = "https://argoproj.github.io/argo-helm"
	ArgocdChartName     = "argo-cd"
	ArgocdChartVersion  = "8.0.0"
	ArgocdReleaseName   = "argocd"
	ArgocdNamespace     = "argocd"
	ArgoRepoName        = "argo"
	ArgocdValuesFileURL = "https://raw.githubusercontent.com/mrgb7/core-infrastructure/" +
		"refs/heads/main/argocd/argocd-values-local.yaml"
)

const (
	HTTPTimeoutSeconds = 30
	MaxResponseSize    = 10 * 1024 * 1024
)

func NewArgocd(kubeConfig string) (*Argocd, error) {
	t, err := NewInstallerTracker(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create installer tracker: %w", err)
	}
	argo := &Argocd{
		KubeConfig: kubeConfig,
		Tracker:    t,
	}
	argo.BasePlugin = NewBasePlugin(kubeConfig, argo)
	return argo, nil
}

func (a *Argocd) GetName() string {
	return "argocd"
}

func (a *Argocd) GetOptions() PluginOptions {
	return PluginOptions{
		Version:          &ArgocdChartVersion,
		Namespace:        &ArgocdNamespace,
		ChartName:        &ArgocdChartName,
		RepoName:         &ArgoRepoName,
		Repository:       &ArgocdRepoURL,
		releaseName:      &ArgocdReleaseName,
		ChartValues:      a.getChartValues(),
		CRDsGroupVersion: "argoproj.io",
	}
}

func (a *Argocd) Install(kubeConfig, clusterName string, ensure ...bool) error {
	return a.UnifiedInstall(kubeConfig, clusterName, ensure...)
}

func (a *Argocd) Uninstall(kubeConfig, clusterName string, ensure ...bool) error {
	if err := a.checkUsage(); err != nil {
		return err
	}

	return a.UnifiedUninstall(kubeConfig, clusterName, ensure...)
}

func (a *Argocd) checkUsage() error {
	plugins, _ := a.Tracker.GetAllPluginByInstaller(a.GetName())

	if len(plugins) > 0 {
		return fmt.Errorf("you cannot uninstall argocd because it is used by other plugins: %v", plugins)
	}
	return nil
}

func (a *Argocd) getValuesContent() (map[string]interface{}, error) {
	if _, err := url.Parse(ArgocdValuesFileURL); err != nil {
		return nil, fmt.Errorf("invalid values file URL: %w", err)
	}

	httpClient := &http.Client{
		Timeout: HTTPTimeoutSeconds * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), HTTPTimeoutSeconds*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", ArgocdValuesFileURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch values file: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Debugln("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch values file: HTTP %d %s", resp.StatusCode, resp.Status)
	}

	limitedReader := io.LimitReader(resp.Body, MaxResponseSize)
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	hash := sha256.Sum256(content)
	logger.Debugf("ArgoCD values file SHA256: %x", hash)

	var values map[string]interface{}
	if err := yaml.Unmarshal(content, &values); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML content: %w", err)
	}

	return values, nil
}

func (a *Argocd) Status() string {
	c, err := k8s.NewK8sClient(a.KubeConfig)
	if err != nil {
		logger.Debugf("failed to create k8s client: %v", err)
		return StatusUnknown
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ns, err := c.GetNameSpace(ArgocdNamespace, ctx)
	if ns == "" || err != nil {
		logger.Debugf("failed to get argocd namespace: %v", err)
		return StatusNotInstalled
	}
	return StatusRunning
}

func (a *Argocd) getChartValues() map[string]interface{} {
	val, err := a.getValuesContent()
	if err != nil {
		logger.Errorln("failed to get values content: %v", err)
		return nil
	}
	return val
}

func (a *Argocd) GetDependencies() []string {
	return []string{}
}
