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
	Long:  `Add a new plugin to the cluster`,
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

				err := plugin.Install(c.KubeConfig, c.Name)
				if err != nil {
					logger.Errorln("Error installing plugin: %v", err)
				} else {
					logger.Successln("Successfully installed %s", pName)
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
	flags := addCmd.Flags()
	flags.StringVarP(&pName, "name", "n", "", "Name of the plugin")
	flags.StringVarP(&cName, "cluster", "c", "", "Name of the cluster")
	addCmd.MarkFlagRequired("name")
	addCmd.MarkFlagRequired("cluster")
	PluginCmd.AddCommand(addCmd)
}
