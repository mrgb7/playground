package plugin

import (
	"fmt"

	"github.com/mrgb7/playground/internal/plugins"
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

		c.SetKubeConfig()

		plugins := []plugins.Plugin{
			&plugins.Argocd{
				KubeConfig: c.KubeConfig,
			},
		}

		for _, plugin := range plugins {
			if plugin.GetName() == pName {
				err := plugin.Uninstall()
				if err != nil {
					fmt.Printf("Error uninstalling plugin: %v\n", err)
				}
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
