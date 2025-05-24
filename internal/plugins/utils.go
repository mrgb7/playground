package plugins

import (
	"context"
	"time"

	"github.com/mrgb7/playground/internal/installer"
	"github.com/mrgb7/playground/internal/k8s"
	"github.com/mrgb7/playground/pkg/logger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ArgocdInstallNamespace = "argocd"
	ArgocdServerLabelSelector = "app.kubernetes.io/name=argocd-server"
)

func IsArgoCDRunning(kubeConfig string) bool {
	client, err := k8s.NewK8sClient(kubeConfig)
	if err != nil {
		logger.Debug("Failed to create k8s client: %v", err)
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	namespace, err := client.GetNameSpace(ArgocdInstallNamespace, ctx)
	if err != nil || namespace == "" {
		logger.Debug("ArgoCD namespace not found: %v", err)
		return false
	}

	podList, err := client.Clientset.CoreV1().Pods(ArgocdInstallNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: ArgocdServerLabelSelector,
	})
	if err != nil {
		logger.Debug("Failed to list ArgoCD server pods: %v", err)
		return false
	}

	if len(podList.Items) == 0 {
		logger.Debug("No ArgoCD server pods found")
		return false
	}

	for _, pod := range podList.Items {
		if pod.Status.Phase == "Running" {
			readyContainers := 0
			totalContainers := len(pod.Status.ContainerStatuses)
			
			for _, containerStatus := range pod.Status.ContainerStatuses {
				if containerStatus.Ready {
					readyContainers++
				}
			}
			
			if readyContainers == totalContainers && totalContainers > 0 {
				return true
			}
		}
	}

	logger.Debug("ArgoCD pods are not ready")
	return false
}

func NewInstaller(plugin Plugin, kubeConfig, clusterName string) (installer.Installer, error) {
	if IsArgoCDRunning(kubeConfig) {
		argoInstaller, err := installer.NewArgoInstaller(kubeConfig, clusterName)
		if err != nil {
			logger.Warn("Failed to create ArgoCD installer, falling back to Helm: %v", err)
			return plugin.GetInstaller()
		}
		
		return argoInstaller, nil
	}

	return plugin.GetInstaller()
}

func NewArgoOptions(plugin Plugin) *installer.InstallOptions {
	pluginName := plugin.GetName()
	
	switch pluginName {
	case "cert-manager":
		return &installer.InstallOptions{
			ApplicationName: "cert-manager-app",
			RepoURL:        "https://github.com/mrgb7/core-infrastructure",
			Path:           "cert-manager",
			TargetRevision: "main",
			Namespace:      "cert-manager",
		}
	case "argocd":
		return &installer.InstallOptions{
			ApplicationName: "argocd-app",
			RepoURL:        "https://github.com/mrgb7/core-infrastructure", 
			Path:           "argocd",
			TargetRevision: "main",
			Namespace:      "argocd",
		}
	case "loadBalancer":
		return &installer.InstallOptions{
			ApplicationName: "metallb-app",
			RepoURL:        "https://github.com/metallb/metallb",
			Path:           "charts/metallb",
			TargetRevision: "main",
			Namespace:      "metallb-system",
		}
	case "nginx":
		return &installer.InstallOptions{
			ApplicationName: "nginx-app",
			RepoURL:        "https://github.com/mrgb7/core-infrastructure",
			Path:           "nginx",
			TargetRevision: "main",
			Namespace:      "nginx-system",
		}
	default:
		return &installer.InstallOptions{
			ApplicationName: pluginName + "-app",
			RepoURL:        "https://github.com/mrgb7/core-infrastructure",
			Path:           pluginName,
			TargetRevision: "main",
			Namespace:      pluginName + "-system",
		}
	}
} 