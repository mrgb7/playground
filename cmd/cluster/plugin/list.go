package plugin

import "github.com/spf13/cobra"

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available plugins",
	Long:  `List all available plugins for the cluster`,
	Run: func(cmd *cobra.Command, args []string) {
		plugins := []string{"loadBalancer", "argoCD", "grafana", "prometheus"}
		for _, plugin := range plugins {
			println(plugin)
		}
	},
}

func init() {
	PluginCmd.AddCommand(listCmd)
}

