package plugin

import (
	"github.com/mrgb7/playground/internal/plugins"
	"github.com/mrgb7/playground/pkg/logger"
	"github.com/mrgb7/playground/types"
	"github.com/spf13/cobra"
)

// getPluginStatusSilently gets plugin status without displaying errors
func getPluginStatusSilently(plugin plugins.Plugin) string {
	// Enable silent mode to suppress error logging
	logger.SetSilentMode(true)
	
	// Get the status 
	status := plugin.Status()
	
	// Restore normal logging
	logger.SetSilentMode(false)
	
	return status
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available plugins",
	Long:  `List all available plugins for the cluster`,
	Run: func(cmd *cobra.Command, args []string) {
		clusterName, _ := cmd.Flags().GetString("cluster-name")
		if clusterName == "" {
			logger.Infoln("Basic plugins:")
			
			for _, plugin := range plugins.List {
				status := getPluginStatusSilently(plugin)
				logger.Infoln("  %s: %s", plugin.GetName(), status)
			}
			
			logger.Infoln("")
			logger.Infoln("For cluster-specific plugins, specify --cluster-name")
			return
		}

		c := types.Cluster{
			Name: clusterName,
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
			status := getPluginStatusSilently(plugin)
			logger.Infoln("  %s: %s", plugin.GetName(), status)
		}
	},
}

func init() {
	listCmd.Flags().StringP("cluster-name", "c", "", "Cluster name to list plugins for")
	PluginCmd.AddCommand(listCmd)
}
