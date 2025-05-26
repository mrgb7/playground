package plugin

import (
	"github.com/spf13/cobra"
)

var PluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage plugins",
	Long:  `Manage plugins for the cluster`,
}

func init() {
}
