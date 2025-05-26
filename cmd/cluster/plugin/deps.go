package plugin

import (
	"github.com/mrgb7/playground/internal/plugins"
	"github.com/mrgb7/playground/pkg/logger"
	"github.com/mrgb7/playground/types"
	"github.com/spf13/cobra"
)

var depsCmd = &cobra.Command{
	Use:   "deps",
	Short: "Show plugin dependencies",
	Long:  `Show dependency information for plugins including dependencies and dependents`,
	Run: func(cmd *cobra.Command, args []string) {
		c := types.Cluster{
			Name: cName,
		}

		ip := c.GetMasterIP()
		if err := c.SetKubeConfig(); err != nil {
			logger.Errorln("Failed to set kubeconfig: %v", err)
			return
		}

		dependencyPlugins, err := plugins.CreateDependencyPluginsList(c.KubeConfig, ip, c.Name)
		if err != nil {
			logger.Errorln("Failed to create dependency plugins list: %v", err)
			return
		}

		validator := plugins.NewDependencyValidator(dependencyPlugins)

		if pName != "" {
			dependencies, dependents := validator.GetDependencyInfo(pName)
			
			logger.Infoln("Plugin: %s", pName)
			if len(dependencies) > 0 {
				logger.Infoln("  Dependencies: %v", dependencies)
			} else {
				logger.Infoln("  Dependencies: none")
			}
			
			if len(dependents) > 0 {
				logger.Infoln("  Dependents: %v", dependents)
			} else {
				logger.Infoln("  Dependents: none")
			}
		} else {
			logger.Infoln("Plugin Dependency Information:")
			logger.Infoln("=============================")
			
			for _, plugin := range dependencyPlugins {
				name := plugin.GetName()
				dependencies, dependents := validator.GetDependencyInfo(name)
				
				logger.Infoln("")
				logger.Infoln("Plugin: %s", name)
				if len(dependencies) > 0 {
					logger.Infoln("  Dependencies: %v", dependencies)
				} else {
					logger.Infoln("  Dependencies: none")
				}
				
				if len(dependents) > 0 {
					logger.Infoln("  Dependents: %v", dependents)
				} else {
					logger.Infoln("  Dependents: none")
				}
			}
		}
	},
}

func init() {
	flags := depsCmd.Flags()
	flags.StringVarP(&pName, "name", "n", "", "Name of the plugin (optional, shows all if not specified)")
	flags.StringVarP(&cName, "cluster", "c", "", "Name of the cluster")
	if err := depsCmd.MarkFlagRequired("cluster"); err != nil {
		logger.Errorln("Failed to mark cluster flag as required: %v", err)
	}
	PluginCmd.AddCommand(depsCmd)
} 