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
	ArgocdInstallNamespace    = "argocd"
	ArgocdServerLabelSelector = "app.kubernetes.io/name=argocd-server"
)

func IsArgoCDRunning(kubeConfig string) bool {
	client, err := k8s.NewK8sClient(kubeConfig)
	if err != nil {
		logger.Debugln("Failed to create k8s client: %v", err)
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	namespace, err := client.GetNameSpace(ArgocdInstallNamespace, ctx)
	if err != nil || namespace == "" {
		logger.Debugln("ArgoCD namespace not found: %v", err)
		return false
	}

	podList, err := client.Clientset.CoreV1().Pods(ArgocdInstallNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: ArgocdServerLabelSelector,
	})
	if err != nil {
		logger.Debugln("Failed to list ArgoCD server pods: %v", err)
		return false
	}

	if len(podList.Items) == 0 {
		logger.Debugln("No ArgoCD server pods found")
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

	logger.Debugln("ArgoCD pods are not ready")
	return false
}

func NewInstaller(plugin Plugin, kubeConfig, clusterName string) (installer.Installer, error) {
	tracker, err := NewInstallerTracker(kubeConfig)
	if err != nil {
		logger.Warnln("Failed to create installer tracker: %v", err)
		// Continue with legacy logic if tracker fails
	} else {
		recordedInstaller, err := tracker.GetPluginInstaller(plugin.GetName())
		if err != nil {
			logger.Warnln("Failed to get recorded installer for plugin %s: %v", plugin.GetName(), err)
		} else if recordedInstaller != "" {
			// Use the recorded installer type
			logger.Infoln("Using recorded installer type '%s' for plugin '%s'", recordedInstaller, plugin.GetName())
			switch recordedInstaller {
			case InstallerTypeArgoCD:
				return installer.NewArgoInstaller(kubeConfig, clusterName)
			case InstallerTypeHelm:
				return installer.NewHelmInstaller(kubeConfig)
			default:
				logger.Warnln("Unknown recorded installer type '%s' for plugin '%s', falling back to logic", recordedInstaller, plugin.GetName())
			}
		}
	}

	if IsArgoCDRunning(kubeConfig) {
		argoInstaller, err := installer.NewArgoInstaller(kubeConfig, clusterName)
		if err != nil {
			logger.Errorln("Failed to create ArgoCD installer: %v", err)
			return nil, err
		}
		return argoInstaller, nil
	}

	return installer.NewHelmInstaller(kubeConfig)
}
