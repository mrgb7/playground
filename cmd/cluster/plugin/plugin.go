package plugin

import (
	"github.com/mrgb7/playground/internal/plugins"
	"github.com/mrgb7/playground/pkg/logger"
	"github.com/mrgb7/playground/types"
	"github.com/spf13/cobra"
)

var PluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage plugins",
	Long:  `Manage plugins for the cluster`,
}

type pluginOperation func(plugin plugins.Plugin, kubeConfig, clusterName string) error

func executePluginOperation(pluginName, clusterName string, operation pluginOperation,
	successMsg, errorMsg string) error {
	c := types.Cluster{
		Name: clusterName,
	}

	ip := c.GetMasterIP()
	if err := c.SetKubeConfig(); err != nil {
		logger.Errorln("Failed to set kubeconfig: %v", err)
		return err
	}

	pluginsList, err := plugins.CreatePluginsList(c.KubeConfig, ip, c.Name)
	if err != nil {
		logger.Errorln("Failed to create plugins list: %v", err)
		return err
	}

	for _, plugin := range pluginsList {
		if plugin.GetName() == pluginName {
			err := operation(plugin, c.KubeConfig, c.Name)
			if err != nil {
				logger.Errorln("%s: %v", errorMsg, err)
				return err
			}
			logger.Successln(successMsg, pluginName)
			return nil
		}
	}

	logger.Errorln("Plugin %s not found", pluginName)
	logger.Infoln("Available plugins:")
	for _, plugin := range pluginsList {
		logger.Infoln("  - %s", plugin.GetName())
	}
	return nil
}

func init() {
}
