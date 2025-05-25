package cluster

import (
	"github.com/mrgb7/playground/internal/multipass"
	"github.com/mrgb7/playground/pkg/logger"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all existing clusters",
	Long:  `List all existing clusters by finding multipass instances ending with '-master'`,
	Run: func(cmd *cobra.Command, args []string) {
		client := multipass.NewMultipassClient()

		if !client.IsMultipassInstalled() {
			logger.Errorln("Error: Multipass is not installed or not in PATH. Please install Multipass first.")
			return
		}

		clusters, err := client.ListClusters()
		if err != nil {
			logger.Errorln("Failed to list clusters: %v", err)
			return
		}

		if len(clusters) == 0 {
			logger.Infoln("No clusters found.")
			return
		}

		logger.Infoln("Available clusters:")
		for _, cluster := range clusters {
			logger.Infoln("  - %s", cluster)
		}
	},
}
