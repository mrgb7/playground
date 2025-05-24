package root

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// Version will be set at build time
	Version = "dev"
	// GitCommit will be set at build time
	GitCommit = "unknown"
	// BuildDate will be set at build time
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of playground",
	Long:  `All software has versions. This is playground's.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("playground version %s\n", Version)
		if verbose, _ := cmd.Flags().GetBool("verbose"); verbose {
			fmt.Printf("Git commit: %s\n", GitCommit)
			fmt.Printf("Build date: %s\n", BuildDate)
			fmt.Printf("Go version: %s\n", runtime.Version())
			fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		}
	},
}

func init() {
	versionCmd.Flags().BoolP("verbose", "v", false, "Show verbose version information")
	rootCmd.AddCommand(versionCmd)
}
