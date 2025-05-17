package plugin

import (
	"github.com/mrgb7/playground/internal/plugins"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a plugin",
	Long:  "Get a plugin for the cluster",
	Run: func(cmd *cobra.Command, args []string) {
		for _, plugin := range plugins.List {
			if len(args) > 0 {
				if plugin.GetName() == args[0] {
					println(plugin.GetName())
				} else {
					println("Plugin not found", args[0])
				}
			}
		}
	},
}

func init() {
	PluginCmd.AddCommand(getCmd)
}
