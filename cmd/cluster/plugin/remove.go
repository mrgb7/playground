package plugin

import (
	"github.com/mrgb7/playground/internal/plugins"
	"github.com/mrgb7/playground/pkg/logger"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "remove plugin",
	Long:  `Remove plugin from the cluster`,
	Run: func(cmd *cobra.Command, args []string) {
		uninstallOperation := func(plugin plugins.Plugin, kubeConfig, clusterName string) error {
			return plugin.Uninstall(kubeConfig, clusterName)
		}

		_ = executePluginOperation(pName, cName, uninstallOperation,
			"Successfully uninstalled %s", "Error uninstalling plugin")
	},
}

func init() {
	flags := removeCmd.Flags()
	flags.StringVarP(&pName, "name", "n", "", "Name of the plugin")
	flags.StringVarP(&cName, "cluster", "c", "", "Name of the cluster")
	if err := removeCmd.MarkFlagRequired("name"); err != nil {
		logger.Errorln("Failed to mark name flag as required: %v", err)
	}
	if err := removeCmd.MarkFlagRequired("cluster"); err != nil {
		logger.Errorln("Failed to mark cluster flag as required: %v", err)
	}
	PluginCmd.AddCommand(removeCmd)
}
