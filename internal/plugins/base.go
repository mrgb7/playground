package plugins

import (
	"fmt"

	"github.com/mrgb7/playground/internal/installer"
	"github.com/mrgb7/playground/pkg/logger"
)

type BasePlugin struct {
	KubeConfig string
	plugin     Plugin
}

func NewBasePlugin(kubeConfig string, plugin Plugin) *BasePlugin {
	return &BasePlugin{
		KubeConfig: kubeConfig,
		plugin:     plugin,
	}
}

func (b *BasePlugin) SmartInstall(kubeConfig, clusterName string, ensure ...bool) error {
	logger.Info("Starting smart installation for plugin: %s", b.plugin.GetName())
	
	smartInstaller, err := GetSmartInstaller(b.plugin, kubeConfig, clusterName)
	if err != nil {
		return fmt.Errorf("failed to get smart installer: %w", err)
	}

	_, isArgoInstaller := smartInstaller.(*installer.ArgoInstaller)
	
	if isArgoInstaller {
		logger.Info("Using ArgoCD installer for %s", b.plugin.GetName())
		options := CreateArgoInstallOptions(b.plugin)
		return smartInstaller.Install(options)
	} else {
		logger.Info("Using Helm installer for %s", b.plugin.GetName())
		return smartInstaller.Install(&installer.InstallOptions{})
	}
}

func (b *BasePlugin) SmartUninstall(kubeConfig, clusterName string, ensure ...bool) error {
	logger.Info("Starting smart uninstallation for plugin: %s", b.plugin.GetName())
	
	smartInstaller, err := GetSmartInstaller(b.plugin, kubeConfig, clusterName)
	if err != nil {
		return fmt.Errorf("failed to get smart installer: %w", err)
	}

	_, isArgoInstaller := smartInstaller.(*installer.ArgoInstaller)
	
	if isArgoInstaller {
		logger.Info("Using ArgoCD installer for %s removal", b.plugin.GetName())
		options := CreateArgoInstallOptions(b.plugin)
		return smartInstaller.UnInstall(options)
	} else {
		logger.Info("Using Helm installer for %s removal", b.plugin.GetName())
		return smartInstaller.UnInstall(&installer.InstallOptions{})
	}
} 