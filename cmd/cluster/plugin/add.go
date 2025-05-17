package plugin

import (
	"fmt"

	"github.com/mrgb7/playground/internal/plugins"
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
		// Get the name of the plugin from the flags
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
				err := plugin.Install()
				if err != nil {
					fmt.Printf("Error installing plugin: %v\n", err)
				}
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
