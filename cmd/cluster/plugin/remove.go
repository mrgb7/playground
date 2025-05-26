package plugin

import (
	"github.com/mrgb7/playground/internal/plugins"
	"github.com/mrgb7/playground/pkg/logger"
	"github.com/mrgb7/playground/types"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "remove plugin",
	Long:  `Remove plugin from the cluster with automatic dependency resolution`,
	Run: func(cmd *cobra.Command, args []string) {
		c := types.Cluster{
			Name: cName,
		}

		ip := c.GetMasterIP()
		if err := c.SetKubeConfig(); err != nil {
			logger.Errorln("Failed to set kubeconfig: %v", err)
			return
		}

		// Validate dependencies and get uninstall order
		uninstallOrder, err := plugins.ValidateAndGetUninstallOrder(pName, c.KubeConfig, ip, c.Name)
		if err != nil {
			logger.Errorln("Dependency validation failed: %v", err)
			return
		}

		logger.Infoln("Plugin uninstallation order: %v", uninstallOrder)

		// Get plugins list
		pluginsList, err := plugins.CreatePluginsList(c.KubeConfig, ip, c.Name)
		if err != nil {
			logger.Errorln("Failed to create plugins list: %v", err)
			return
		}

		// Create plugin map for quick lookup
		pluginMap := make(map[string]plugins.Plugin)
		for _, plugin := range pluginsList {
			pluginMap[plugin.GetName()] = plugin
		}

		// Uninstall plugins in dependency order
		for _, pluginName := range uninstallOrder {
			plugin, exists := pluginMap[pluginName]
			if !exists {
				logger.Errorln("Plugin %s not found", pluginName)
				return
			}

			// Check if plugin is actually installed
			status := plugin.Status()
			if !plugins.IsPluginInstalled(status) {
				logger.Infoln("Plugin %s is not installed, skipping", pluginName)
				continue
			}

			logger.Infoln("Uninstalling plugin: %s", pluginName)
			err := plugin.Uninstall(c.KubeConfig, c.Name)
			if err != nil {
				logger.Errorln("Error uninstalling plugin %s: %v", pluginName, err)
				return
			}
			logger.Successln("Successfully uninstalled %s", pluginName)
		}

		logger.Successln("All plugins uninstalled successfully!")
	},
}

func init() {
	flags := removeCmd.Flags()
	flags.StringVarP(&pName, "name", "n", "", "Name of the plugin")
	flags.StringVarP(&cName, "cluster", "c", "", "Name of the cluster")
	if err := removeCmd.MarkFlagRequired("name"); err != nil {
		logger.Errorln("Failed to mark name flag as required: %v", err)
	}
	if err := removeCmd.MarkFlagRequired("cluster"); err != nil {
		logger.Errorln("Failed to mark cluster flag as required: %v", err)
	}
	PluginCmd.AddCommand(removeCmd)
}
