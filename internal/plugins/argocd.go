package plugins

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/mrgb7/playground/internal/installer"
	"github.com/mrgb7/playground/internal/k8s"
	"github.com/mrgb7/playground/pkg/logger"
	"gopkg.in/yaml.v3"
)

type Argocd struct {
	KubeConfig string
	*BasePlugin
}

const (
	ArgocdRepoUrl       = "https://argoproj.github.io/argo-helm"
	ArgocdChartName     = "argo-cd"
	ArgocdChartVersion  = "8.0.0"
	ArgocdReleaseName   = "argocd"
	ArgocdNamespace     = "argocd"
	ArgoRepoName        = "argo"
	ArgocdValuesFileURL = "https://raw.githubusercontent.com/mrgb7/core-infrastructure/refs/heads/main/argocd/argocd-values-local.yaml"
	
	HTTPTimeoutSeconds = 30
	MaxResponseSize    = 10 * 1024 * 1024
)

func NewArgocd(kubeConfig string) *Argocd {
	argo := &Argocd{
		KubeConfig: kubeConfig,
	}
	argo.BasePlugin = NewBasePlugin(kubeConfig, argo)
	return argo
}

func (a *Argocd) GetName() string {
	return "argocd"
}

func (a *Argocd) Install(ensure ...bool) error {
	inst, err := NewInstaller(a, a.KubeConfig, "")
	if err != nil {
		return fmt.Errorf("failed to get installer from factory: %w", err)
	}

	_, isArgo := inst.(*installer.ArgoInstaller)
	
	if isArgo {
		options := NewArgoOptions(a)
		err = inst.Install(options)
	} else {
		err = inst.Install(&installer.InstallOptions{})
	}
	
	if err != nil {
		return fmt.Errorf("failed to install argocd: %w", err)
	}
	return nil
}

func (a *Argocd) Uninstall(ensure ...bool) error {
	inst, err := NewInstaller(a, a.KubeConfig, "")
	if err != nil {
		return fmt.Errorf("failed to get installer from factory: %w", err)
	}

	_, isArgo := inst.(*installer.ArgoInstaller)
	
	if isArgo {
		options := NewArgoOptions(a)
		err = inst.UnInstall(options)
	} else {
		err = inst.UnInstall(&installer.InstallOptions{})
	}
	
	if err != nil {
		return fmt.Errorf("failed to uninstall argocd: %w", err)
	}
	return nil
}

func (a *Argocd) GetInstaller() (installer.Installer, error) {
	val, err := a.getValuesContent()
	if err != nil {
		return nil, fmt.Errorf("failed to get values content: %w", err)
	}
	return &installer.HelmInstaller{
		ReleaseName:  ArgocdReleaseName,
		ChartName:    ArgocdChartName,
		RepoUrl:      ArgocdRepoUrl,
		RepoName:     ArgoRepoName,
		Namespace:    ArgocdNamespace,
		ChartVersion: ArgocdChartVersion,
		Values:       val,
		KubeConfig:   a.KubeConfig,
	}, nil
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
	defer resp.Body.Close()
	
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
		logger.Error("failed to create k8s client: %v", err)
		return "UNKOWN"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ns, err := c.GetNameSpace(ArgocdNamespace, ctx)
	if ns == "" || err != nil {
		logger.Error("failed to get argocd namespace: %v", err)
		return "Not installed"
	}
	return "argocd is running"
}

func (a *Argocd) FactoryInstall(kubeConfig, clusterName string, ensure ...bool) error {
	return a.BasePlugin.FactoryInstall(kubeConfig, clusterName, ensure...)
}

func (a *Argocd) FactoryUninstall(kubeConfig, clusterName string, ensure ...bool) error {
	return a.BasePlugin.FactoryUninstall(kubeConfig, clusterName, ensure...)
}
