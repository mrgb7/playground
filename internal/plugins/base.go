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

func (b *BasePlugin) InstallWithFactory(kubeConfig, clusterName string, ensure ...bool) error {
	logger.Info("Starting factory-based installation for plugin: %s", b.plugin.GetName())
	
	installerFactory, err := CreateInstaller(b.plugin, kubeConfig, clusterName)
	if err != nil {
		return fmt.Errorf("failed to create installer from factory: %w", err)
	}

	_, isArgoInstaller := installerFactory.(*installer.ArgoInstaller)
	
	if isArgoInstaller {
		logger.Info("Using ArgoCD installer for %s", b.plugin.GetName())
		options := CreateArgoInstallOptions(b.plugin)
		return installerFactory.Install(options)
	} else {
		logger.Info("Using Helm installer for %s", b.plugin.GetName())
		return installerFactory.Install(&installer.InstallOptions{})
	}
}

func (b *BasePlugin) UninstallWithFactory(kubeConfig, clusterName string, ensure ...bool) error {
	logger.Info("Starting factory-based uninstallation for plugin: %s", b.plugin.GetName())
	
	installerFactory, err := CreateInstaller(b.plugin, kubeConfig, clusterName)
	if err != nil {
		return fmt.Errorf("failed to create installer from factory: %w", err)
	}

	_, isArgoInstaller := installerFactory.(*installer.ArgoInstaller)
	
	if isArgoInstaller {
		logger.Info("Using ArgoCD installer for %s removal", b.plugin.GetName())
		options := CreateArgoInstallOptions(b.plugin)
		return installerFactory.UnInstall(options)
	} else {
		logger.Info("Using Helm installer for %s removal", b.plugin.GetName())
		return installerFactory.UnInstall(&installer.InstallOptions{})
	}
} 