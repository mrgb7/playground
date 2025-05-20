package cluster

import (
	"sync"

	"github.com/mrgb7/playground/internal/multipass"
	"github.com/mrgb7/playground/pkg/logger"
	"github.com/spf13/cobra"
)

var (
	cPurge bool
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean up cluster resources",
	Long:  `Clean up cluster resources, including stopping and removing nodes`,
	Run: func(cmd *cobra.Command, args []string) {
		var wg sync.WaitGroup
		client := multipass.NewMultipassClient()

		if !client.IsMultipassInstalled() {
			logger.Errorln("Error: Multipass is not installed or not in PATH. Please install Multipass first.")
			return
		}

		if len(args) > 0 {
			clusterName := args[0]
			logger.Infoln("Cleaning up resources for cluster '%s'...", clusterName)

			// Delete the cluster
			if err := client.DeleteCluster(clusterName, &wg); err != nil {
				logger.Errorln("Failed to clean up cluster: %v", err)
				return
			}
			wg.Wait() // Wait for all goroutines to complete

			logger.Successln("Successfully cleaned up cluster '%s'", clusterName)
		}

		// If purge flag is set or no cluster name was provided, purge all deleted instances
		if cPurge || len(args) == 0 {
			logger.Infoln("Purging all deleted instances...")
			if err := client.PurgeNodes(); err != nil {
				logger.Errorln("Failed to purge deleted instances: %v", err)
				return
			}
			logger.Successln("Successfully purged all deleted instances")
		}
	},
}

func init() {
	ClusterCmd.AddCommand(cleanCmd)
	cleanCmd.Flags().BoolVarP(&cPurge, "purge", "p", false, "Purge all resources")
}
