package plugins

import (
	"fmt"

	"github.com/mrgb7/playground/internal/installer"
)

const (
	StatusNotInstalled = "Not installed"
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

func (b *BasePlugin) UnifiedInstall(kubeConfig, clusterName string, ensure ...bool) error {
	inst, err := NewInstaller(b.plugin, kubeConfig, clusterName)
	if err != nil {
		return fmt.Errorf("failed to create installer: %w", err)
	}

	opts := newInstallOptions(b.plugin, kubeConfig)

	return inst.Install(opts)
}

func (b *BasePlugin) UnifiedUninstall(kubeConfig, clusterName string, ensure ...bool) error {
	inst, err := NewInstaller(b.plugin, kubeConfig, clusterName)
	if err != nil {
		return fmt.Errorf("failed to create installer: %w", err)
	}
	opts := newInstallOptions(b.plugin, kubeConfig)
	return inst.UnInstall(opts)
}

func newInstallOptions(plugin Plugin, kubeConfig string) *installer.InstallOptions {
	chartName := plugin.GetChartName()
	version := plugin.GetVersion()
	return &installer.InstallOptions{
		Namespace:       plugin.GetNamespace(),
		Values:          plugin.GetChartValues(),
		ChartName:       &chartName,
		RepoURL:         plugin.GetRepository(),
		ApplicationName: plugin.GetName(),
		Version:         version,
		KubeConfig:      kubeConfig,
		RepoName:        plugin.GetRepoName(),
	}
}
