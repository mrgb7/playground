package plugins

import (
	"fmt"

	"github.com/mrgb7/playground/internal/installer"
	"github.com/mrgb7/playground/pkg/logger"
)

const (
	StatusNotInstalled = "Not installed"
	StatusUnknown      = "UNKNOWN"
	StatusRunning      = "running"
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

	// Determine installer type for tracking
	var installerType string
	switch inst.(type) {
	case *installer.ArgoInstaller:
		installerType = InstallerTypeArgoCD
	case *installer.HelmInstaller:
		installerType = InstallerTypeHelm
	default:
		installerType = "unknown"
	}

	opts := newInstallOptions(b.plugin, kubeConfig)

	// Install the plugin
	err = inst.Install(opts)
	if err != nil {
		return err
	}

	tracker, trackerErr := NewInstallerTracker(kubeConfig)
	if trackerErr != nil {
		logger.Warnln("Failed to create installer tracker after installing %s: %v", b.plugin.GetName(), trackerErr)
	} else {
		recordErr := tracker.RecordPluginInstaller(b.plugin.GetName(), installerType)
		if recordErr != nil {
			logger.Warnln("Failed to record installer type for %s: %v", b.plugin.GetName(), recordErr)
		}
	}

	return nil
}

func (b *BasePlugin) UnifiedUninstall(kubeConfig, clusterName string, ensure ...bool) error {
	inst, err := NewInstaller(b.plugin, kubeConfig, clusterName)
	if err != nil {
		return fmt.Errorf("failed to create installer: %w", err)
	}
	opts := newInstallOptions(b.plugin, kubeConfig)

	// Uninstall the plugin
	err = inst.UnInstall(opts)
	if err != nil {
		return err
	}

	tracker, trackerErr := NewInstallerTracker(kubeConfig)
	if trackerErr != nil {
		logger.Warnln("Failed to create installer tracker after uninstalling %s: %v", b.plugin.GetName(), trackerErr)
	} else {
		removeErr := tracker.RemovePluginInstaller(b.plugin.GetName())
		if removeErr != nil {
			logger.Warnln("Failed to remove installer tracking for %s: %v", b.plugin.GetName(), removeErr)
		}
	}

	return nil
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
		// Plugin:          plugin,
	}
}
