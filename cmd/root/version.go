package root

import (
	"github.com/mrgb7/playground/pkg/logger"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of the application",
	Long:  `All software has versions. This is playground's`,
	Run: func(cmd *cobra.Command, args []string) {
		logger.Successln("Playground CLI v1.0.0")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
