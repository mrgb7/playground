package plugin

import (
	"github.com/mrgb7/playground/internal/plugins"
	"github.com/mrgb7/playground/pkg/logger"
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
		installOperation := func(plugin plugins.Plugin, kubeConfig, clusterName string) error {
			return plugin.Install(kubeConfig, clusterName)
		}

		_ = executePluginOperation(pName, cName, installOperation,
			"Successfully installed %s", "Error installing plugin")
	},
}

func init() {
	flags := addCmd.Flags()
	flags.StringVarP(&pName, "name", "n", "", "Name of the plugin")
	flags.StringVarP(&cName, "cluster", "c", "", "Name of the cluster")
	if err := addCmd.MarkFlagRequired("name"); err != nil {
		logger.Errorln("Failed to mark name flag as required: %v", err)
	}
	if err := addCmd.MarkFlagRequired("cluster"); err != nil {
		logger.Errorln("Failed to mark cluster flag as required: %v", err)
	}
	PluginCmd.AddCommand(addCmd)
}
