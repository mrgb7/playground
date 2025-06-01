package plugin

import (
	"github.com/mrgb7/playground/internal/plugins"
	"github.com/mrgb7/playground/pkg/logger"
	"github.com/mrgb7/playground/types"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available plugins",
	Long:  `List all available plugins for the cluster`,
	Run: func(cmd *cobra.Command, args []string) {
		clusterName, _ := cmd.Flags().GetString("cluster-name")

		c := types.Cluster{
			Name: clusterName,
		}

		if !c.IsExists() {
			logger.Errorln("Cluster '%s' does not exist. Please create it first.", clusterName)
			return
		}

		ip := c.GetMasterIP()
		if err := c.SetKubeConfig(); err != nil {
			logger.Errorln("Failed to set kubeconfig: %v", err)
			return
		}

		pluginsList, err := plugins.CreatePluginsList(c.KubeConfig, ip, c.Name)
		if err != nil {
			logger.Errorln("Failed to create plugins list: %v", err)
			return
		}

		logger.Infoln("Available plugins for cluster '%s':", clusterName)

		for _, plugin := range pluginsList {
			status := plugin.Status()
			logger.Infoln("  %s: %s", plugin.GetName(), status)
		}
	},
}

func init() {
	listCmd.Flags().StringP("cluster-name", "c", "", "Cluster name to list plugins for")
	PluginCmd.AddCommand(listCmd)
}
