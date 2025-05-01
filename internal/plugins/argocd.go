package plugins

import (
	"fmt"

	"github.com/mohamedragab2024/playground/internal/helm"
)

type Argocd struct{}

func (a *Argocd) GetName() string {
	return "argocd"
}

func (a *Argocd) Install(kubeConfig string, ensure ...bool) error {
	hc := helm.NewClient(kubeConfig, "argocd", true)
	opt := helm.InstallOptions{
		ReleaseName: "argocd",
		ChartName:   "argo/argo-cd",
		RepoName:    "argo",
		RepoURL:     "https://argoproj.github.io/argo-helm",
	}
	err := hc.Install(opt)
	if err != nil {
		return fmt.Errorf("failed to install argocd: %w", err)
	}
	fmt.Println("Argocd installed successfully")
	return nil
}

func (a *Argocd) Uninstall(kubeConfig string, ensure ...bool) error {
	fmt.Println("Uninstalling argocd")
	return nil
}

func (a *Argocd) GetValuesContent() (string, error) {
	return "", nil
}
