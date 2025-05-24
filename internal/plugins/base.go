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

func (b *BasePlugin) FactoryInstall(kubeConfig, clusterName string, ensure ...bool) error {
	logger.Info("Starting factory-based installation for plugin: %s", b.plugin.GetName())
	
	inst, err := NewInstaller(b.plugin, kubeConfig, clusterName)
	if err != nil {
		return fmt.Errorf("failed to create installer from factory: %w", err)
	}

	_, isArgo := inst.(*installer.ArgoInstaller)
	
	if isArgo {
		logger.Info("Using ArgoCD installer for %s", b.plugin.GetName())
		options := NewArgoOptions(b.plugin)
		return inst.Install(options)
	} else {
		logger.Info("Using Helm installer for %s", b.plugin.GetName())
		return inst.Install(&installer.InstallOptions{})
	}
}

func (b *BasePlugin) FactoryUninstall(kubeConfig, clusterName string, ensure ...bool) error {
	logger.Info("Starting factory-based uninstallation for plugin: %s", b.plugin.GetName())
	
	inst, err := NewInstaller(b.plugin, kubeConfig, clusterName)
	if err != nil {
		return fmt.Errorf("failed to create installer from factory: %w", err)
	}

	_, isArgo := inst.(*installer.ArgoInstaller)
	
	if isArgo {
		logger.Info("Using ArgoCD installer for %s removal", b.plugin.GetName())
		options := NewArgoOptions(b.plugin)
		return inst.UnInstall(options)
	} else {
		logger.Info("Using Helm installer for %s removal", b.plugin.GetName())
		return inst.UnInstall(&installer.InstallOptions{})
	}
} 