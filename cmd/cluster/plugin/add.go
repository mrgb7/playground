package plugin

import (
	"github.com/mrgb7/playground/internal/plugins"
	"github.com/mrgb7/playground/pkg/logger"
	"github.com/mrgb7/playground/types"
	"github.com/spf13/cobra"
)

var (
	pName string
	cName string
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new plugin",
	Long:  `Add a new plugin to the cluster with automatic dependency resolution`,
	Run: func(cmd *cobra.Command, args []string) {
		c := types.Cluster{
			Name: cName,
		}

		ip := c.GetMasterIP()
		if err := c.SetKubeConfig(); err != nil {
			logger.Errorln("Failed to set kubeconfig: %v", err)
			return
		}

		installOrder, err := plugins.ValidateAndGetInstallOrder(pName, c.KubeConfig, ip, c.Name)
		if err != nil {
			logger.Errorln("Dependency validation failed: %v", err)
			return
		}

		logger.Infoln("Plugin installation order: %v", installOrder)

		pluginsList, err := plugins.CreatePluginsList(c.KubeConfig, ip, c.Name)
		if err != nil {
			logger.Errorln("Failed to create plugins list: %v", err)
			return
		}

		pluginMap := make(map[string]plugins.Plugin)
		for _, plugin := range pluginsList {
			pluginMap[plugin.GetName()] = plugin
		}

		for _, pluginName := range installOrder {
			plugin, exists := pluginMap[pluginName]
			if !exists {
				logger.Errorln("Plugin %s not found", pluginName)
				return
			}
			status := plugin.Status()
			if plugins.IsPluginInstalled(status) {
				logger.Infoln("Plugin %s is already installed, skipping", pluginName)
				continue
			}

			logger.Infoln("Installing plugin: %s", pluginName)
			err := plugin.Install(c.KubeConfig, c.Name)
			if err != nil {
				logger.Errorln("Error installing plugin %s: %v", pluginName, err)
				return
			}
			logger.Successln("Successfully installed %s", pluginName)
		}

		logger.Successln("All plugins installed successfully!")
	},
}

func init() {
	flags := addCmd.Flags()
	flags.StringVarP(&pName, "name", "n", "", "Name of the plugin")
	flags.StringVarP(&cName, "cluster", "c", "", "Name of the cluster")
	if err := addCmd.MarkFlagRequired("name"); err != nil {
		logger.Errorln("Failed to mark name flag as required: %v", err)
	}
	if err := addCmd.MarkFlagRequired("cluster"); err != nil {
		logger.Errorln("Failed to mark cluster flag as required: %v", err)
	}
	PluginCmd.AddCommand(addCmd)
}
