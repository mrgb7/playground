package plugin

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mrgb7/playground/internal/plugins"
	"github.com/mrgb7/playground/pkg/logger"
	"github.com/mrgb7/playground/types"
	"github.com/spf13/cobra"
)

var (
	pName     string
	cName     string
	override  bool
	setValues []string
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new plugin",
	Long:  `Add a new plugin to the cluster with automatic dependency resolution`,
	Run: func(cmd *cobra.Command, args []string) {
		c := types.Cluster{
			Name: cName,
		}

		ip := c.GetMasterIP()
		if err := c.SetKubeConfig(); err != nil {
			logger.Errorln("Failed to set kubeconfig: %v", err)
			return
		}

		// Parse override values if provided
		var overrideValues map[string]interface{}
		if override {
			var err error
			overrideValues, err = parseSetValues(setValues)
			if err != nil {
				logger.Errorln("Failed to parse --set values: %v", err)
				return
			}
			logger.Infoln("Override mode enabled with values: %v", overrideValues)
		}

		installOrder, err := plugins.ValidateAndGetInstallOrder(pName, c.KubeConfig, ip, c.Name)
		if err != nil {
			logger.Errorln("Dependency validation failed: %v", err)
			return
		}

		logger.Infoln("Plugin installation order: %v", installOrder)

		pluginsList, err := plugins.CreatePluginsList(c.KubeConfig, ip, c.Name)
		if err != nil {
			logger.Errorln("Failed to create plugins list: %v", err)
			return
		}

		pluginMap := make(map[string]plugins.Plugin)
		for _, plugin := range pluginsList {
			pluginMap[plugin.GetName()] = plugin
		}

		for _, pluginName := range installOrder {
			plugin, exists := pluginMap[pluginName]
			if !exists {
				logger.Errorln("Plugin %s not found", pluginName)
				return
			}

			// Handle override mode
			if override && pluginName == pName {
				if err := handlePluginOverride(plugin, overrideValues, c.KubeConfig, c.Name); err != nil {
					logger.Errorln("Error overriding plugin %s: %v", pluginName, err)
					return
				}
				logger.Successln("Successfully overridden %s", pluginName)
				continue
			}

			// Normal installation logic
			status := plugin.Status()
			if plugins.IsPluginInstalled(status) {
				continue
			}

			logger.Infoln("Installing plugin: %s", pluginName)
			err := plugin.Install(c.KubeConfig, c.Name, true)
			if err != nil {
				logger.Errorln("Error installing plugin %s: %v", pluginName, err)
				return
			}
			logger.Successln("Successfully installed %s", pluginName)
		}

		logger.Successln("All plugins processed successfully!")
	},
}

func handlePluginOverride(plugin plugins.Plugin, overrideValues map[string]interface{}, kubeConfig, clusterName string) error {
	// Validate if plugin supports override values
	if err := validatePluginOverride(plugin, overrideValues); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Set override values on the plugin if it supports it
	if overridable, ok := plugin.(plugins.OverridablePlugin); ok {
		overridable.SetOverrideValues(overrideValues)
	}

	// Force re-installation
	logger.Infoln("Force re-installing plugin with override values: %s", plugin.GetName())
	return plugin.Install(kubeConfig, clusterName, true)
}

func validatePluginOverride(plugin plugins.Plugin, overrideValues map[string]interface{}) error {
	// Check if plugin supports override
	if validator, ok := plugin.(plugins.OverrideValidator); ok {
		return validator.ValidateOverrideValues(overrideValues)
	}

	// For plugins that don't implement validation, reject override
	return fmt.Errorf("plugin %s does not support override values", plugin.GetName())
}

func parseSetValues(setValues []string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for _, setValue := range setValues {
		parts := strings.SplitN(setValue, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --set format: %s (expected key=value)", setValue)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if key == "" {
			return nil, fmt.Errorf("empty key in --set: %s", setValue)
		}

		// Parse value type (string, bool, number)
		parsedValue := parseValue(value)

		// Handle nested keys (e.g., "server.replicas=3")
		setNestedValue(result, key, parsedValue)
	}

	return result, nil
}

func parseValue(value string) interface{} {
	// Try to parse as boolean
	if value == "true" {
		return true
	}
	if value == "false" {
		return false
	}

	// Try to parse as integer
	if intVal, err := strconv.Atoi(value); err == nil {
		return intVal
	}

	// Try to parse as float
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal
	}

	// Default to string
	return value
}

func setNestedValue(result map[string]interface{}, key string, value interface{}) {
	keys := strings.Split(key, ".")
	current := result

	// Navigate to the nested map, creating maps as needed
	for i, k := range keys[:len(keys)-1] {
		if _, exists := current[k]; !exists {
			current[k] = make(map[string]interface{})
		}
		if nested, ok := current[k].(map[string]interface{}); ok {
			current = nested
		} else {
			// If key exists but is not a map, create a new path
			newPath := strings.Join(keys[:i+2], ".")
			result[newPath] = value
			return
		}
	}

	// Set the final value
	finalKey := keys[len(keys)-1]
	current[finalKey] = value
}

func init() {
	flags := addCmd.Flags()
	flags.StringVarP(&pName, "name", "n", "", "Name of the plugin")
	flags.StringVarP(&cName, "cluster", "c", "", "Name of the cluster")
	flags.BoolVar(&override, "override", false, "Force re-installation/update even if plugin is already installed")
	flags.StringArrayVar(&setValues, "set", []string{}, "Set values for plugin (can be used multiple times)")

	if err := addCmd.MarkFlagRequired("name"); err != nil {
		logger.Errorln("Failed to mark name flag as required: %v", err)
	}
	if err := addCmd.MarkFlagRequired("cluster"); err != nil {
		logger.Errorln("Failed to mark cluster flag as required: %v", err)
	}
	PluginCmd.AddCommand(addCmd)
}
