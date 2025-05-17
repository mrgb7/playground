package root

import (
	"os"

	"github.com/mrgb7/playground/cmd/cluster"
	"github.com/mrgb7/playground/pkg/logger"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "playground",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application.`,
	Run: func(cmd *cobra.Command, args []string) {
		logger.Infoln("Hello from playground CLI!")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logger.Errorln("%v", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.AddCommand(cluster.ClusterCmd)
}
