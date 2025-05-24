package plugins

import (
	"fmt"

	"github.com/mrgb7/playground/internal/installer"
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
	inst, err := NewInstaller(b.plugin, kubeConfig, clusterName)
	if err != nil {
		return fmt.Errorf("failed to create installer from factory: %w", err)
	}

	_, isArgo := inst.(*installer.ArgoInstaller)
	
	if isArgo {
		options := NewArgoOptions(b.plugin)
		return inst.Install(options)
	} else {
		return inst.Install(&installer.InstallOptions{})
	}
}

func (b *BasePlugin) FactoryUninstall(kubeConfig, clusterName string, ensure ...bool) error {
	inst, err := NewInstaller(b.plugin, kubeConfig, clusterName)
	if err != nil {
		return fmt.Errorf("failed to create installer from factory: %w", err)
	}

	_, isArgo := inst.(*installer.ArgoInstaller)
	
	if isArgo {
		options := NewArgoOptions(b.plugin)
		return inst.UnInstall(options)
	} else {
		return inst.UnInstall(&installer.InstallOptions{})
	}
} 