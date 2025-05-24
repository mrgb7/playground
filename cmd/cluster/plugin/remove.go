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
	Long:  `Remove plugin from the cluster`,
	Run: func(cmd *cobra.Command, args []string) {
		c := types.Cluster{
			Name: cName,
		}

		ip := c.GetMasterIP()
		c.SetKubeConfig()

		pluginsList, err := plugins.CreatePluginsList(c.KubeConfig, ip)
		if err != nil {
			logger.Error("Failed to create plugins list: %v", err)
			return
		}

		found := false
		for _, plugin := range pluginsList {
			if plugin.GetName() == pName {
				found = true
				
				if factory, ok := plugin.(plugins.Factory); ok {
					err := factory.FactoryUninstall(c.KubeConfig, c.Name)
					if err != nil {
						logger.Error("Error uninstalling plugin with factory installer: %v", err)
						logger.Info("Falling back to regular uninstallation...")
						err = plugin.Uninstall()
						if err != nil {
							logger.Error("Error uninstalling plugin with regular installer: %v", err)
						} else {
							logger.Info("Successfully uninstalled %s", pName)
						}
					} else {
						logger.Info("Successfully uninstalled %s", pName)
					}
				} else {
					logger.Info("Using regular uninstallation for plugin: %s", pName)
					err := plugin.Uninstall()
					if err != nil {
						logger.Error("Error uninstalling plugin: %v", err)
					} else {
						logger.Info("Successfully uninstalled %s", pName)
					}
				}
				break
			}
		}
		
		if !found {
			logger.Error("Plugin %s not found", pName)
			logger.Info("Available plugins:")
			for _, plugin := range pluginsList {
				logger.Info("  - %s", plugin.GetName())
			}
		}
	},
}

func init() {
	flags := removeCmd.Flags()
	flags.StringVarP(&pName, "name", "n", "", "Name of the plugin")
	flags.StringVarP(&cName, "cluster", "c", "", "Name of the cluster")
	removeCmd.MarkFlagRequired("name")
	removeCmd.MarkFlagRequired("cluster")
	PluginCmd.AddCommand(removeCmd)
}
