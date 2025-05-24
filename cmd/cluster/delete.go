package cluster

import (
	"sync"

	"github.com/mrgb7/playground/internal/multipass"
	"github.com/mrgb7/playground/pkg/logger"
	"github.com/spf13/cobra"
)

var (
	cDeleteForce bool
	deleteCmd    = &cobra.Command{
		Use:   "delete",
		Short: "Delete an existing cluster",
		Long:  `Delete an existing cluster by specifying its name`,
		Run: func(cmd *cobra.Command, args []string) {
			var wg sync.WaitGroup
			if len(args) < 1 {
				logger.Errorln("Error: Cluster name is required")
				cmd.Help()
				return
			}

			clusterToDelete := args[0]

			client := multipass.NewMultipassClient()

			if !client.IsMultipassInstalled() {
				logger.Errorln("Error: Multipass is not installed or not in PATH. Please install Multipass first.")
				return
			}

			if clusterToDelete == "" {
				logger.Errorln("Error: Please provide a valid cluster name to delete.")
				return
			}
			if err := client.DeleteCluster(clusterToDelete, &wg); err != nil {
				logger.Errorln("Failed to delete cluster: %v", err)
				return
			}
			wg.Wait() // Wait for all goroutines to complete

			if cDeleteForce {
				logger.Infoln("Purging deleted instances...")
				if err := client.PurgeNodes(); err != nil {
					logger.Errorln("Failed to purge deleted instances: %v", err)
					return
				}

			}

			logger.Successln("Successfully deleted cluster '%s'", clusterToDelete)
		},
	}
)

func init() {
	deleteCmd.Flags().BoolVarP(&cDeleteForce, "force", "f", false, "Force delete cluster resources")
}
