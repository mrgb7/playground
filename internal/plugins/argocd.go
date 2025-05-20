package plugins

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mrgb7/playground/internal/installer"
	"github.com/mrgb7/playground/internal/k8s"
	"github.com/mrgb7/playground/pkg/logger"
	"gopkg.in/yaml.v3"
)

type Argocd struct {
	KubeConfig string
}

const (
	ArgocdRepoUrl       = "https://argoproj.github.io/argo-helm"
	ArgocdChartName     = "argo-cd"
	ArgocdChartVersion  = "8.0.0"
	ArgocdReleaseName   = "argocd"
	ArgocdNamespace     = "argocd"
	ArgoRepoName        = "argo"
	ArgocdValuesFileURL = "https://raw.githubusercontent.com/mrgb7/core-infrastructure/refs/heads/main/argocd/argocd-values-local.yaml"
)

func (a *Argocd) GetName() string {
	return "argocd"
}

func (a *Argocd) Install(ensure ...bool) error {
	i, err := a.GetInstaller()
	if err != nil {
		return fmt.Errorf("failed to get installer: %w", err)
	}
	err = i.Install(&installer.InstallOptions{})
	if err != nil {
		return fmt.Errorf("failed to install argocd: %w", err)
	}
	return nil
}

func (a *Argocd) Uninstall(ensure ...bool) error {
	i, err := a.GetInstaller()
	if err != nil {
		return fmt.Errorf("failed to get installer: %w", err)
	}
	err = i.UnInstall(&installer.InstallOptions{})
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
	httpClient := &http.Client{}
	resp, err := httpClient.Get(ArgocdValuesFileURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get file content: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get file content: %s", resp.Status)
	}
	res, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}
	content := string(res)
	vl := make(map[string]interface{})
	err = yaml.Unmarshal([]byte(content), &vl)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml: %w", err)
	}
	return vl, nil
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
