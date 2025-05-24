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
			logger.Errorln("Failed to create plugins list: %v", err)
			return
		}

		found := false
		for _, plugin := range pluginsList {
			if plugin.GetName() == pName {
				found = true
				
				err := plugin.Uninstall(c.KubeConfig, c.Name)
				if err != nil {
					logger.Errorln("Error uninstalling plugin: %v", err)
				} else {
					logger.Successln("Successfully uninstalled %s", pName)
				}
				break
			}
		}
		
		if !found {
			logger.Errorln("Plugin %s not found", pName)
			logger.Infoln("Available plugins:")
			for _, plugin := range pluginsList {
				logger.Infoln("  - %s", plugin.GetName())
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
