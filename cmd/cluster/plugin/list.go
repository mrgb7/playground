package plugin

import (
	"github.com/mrgb7/playground/internal/plugins"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available plugins",
	Long:  `List all available plugins for the cluster`,
	Run: func(cmd *cobra.Command, args []string) {
		for _, plugin := range plugins.List {
			println(plugin.GetName(), plugin.Status())
		}
	},
}

func init() {
	PluginCmd.AddCommand(listCmd)
}
