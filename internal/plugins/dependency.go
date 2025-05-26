package plugins

import (
	"fmt"
	"strings"

	"github.com/mrgb7/playground/pkg/logger"
)

// DependencyPlugin extends the Plugin interface to include dependency management
type DependencyPlugin interface {
	Plugin
	GetDependencies() []string
}

// GraphNode represents a node in the dependency graph
type GraphNode struct {
	Plugin       DependencyPlugin
	Dependencies []string // plugins this node depends on
	Dependents   []string // plugins that depend on this node
}

// DependencyGraph represents the plugin dependency graph
type DependencyGraph struct {
	nodes map[string]*GraphNode
}

// NewDependencyGraph creates a new dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes: make(map[string]*GraphNode),
	}
}

// AddPlugin adds a plugin to the dependency graph
func (dg *DependencyGraph) AddPlugin(plugin DependencyPlugin) {
	name := plugin.GetName()
	dependencies := plugin.GetDependencies()
	
	// Create or update the node for this plugin
	if dg.nodes[name] == nil {
		dg.nodes[name] = &GraphNode{
			Plugin:       plugin,
			Dependencies: make([]string, len(dependencies)),
			Dependents:   make([]string, 0),
		}
	} else {
		dg.nodes[name].Plugin = plugin
		dg.nodes[name].Dependencies = make([]string, len(dependencies))
	}
	
	copy(dg.nodes[name].Dependencies, dependencies)
	
	// Update dependent relationships
	for _, dep := range dependencies {
		// Ensure dependency node exists
		if dg.nodes[dep] == nil {
			dg.nodes[dep] = &GraphNode{
				Plugin:       nil, // Will be set when the actual plugin is added
				Dependencies: make([]string, 0),
				Dependents:   make([]string, 0),
			}
		}
		
		// Add this plugin as a dependent of the dependency
		// Check if not already added to avoid duplicates
		found := false
		for _, dependent := range dg.nodes[dep].Dependents {
			if dependent == name {
				found = true
				break
			}
		}
		if !found {
			dg.nodes[dep].Dependents = append(dg.nodes[dep].Dependents, name)
		}
	}
}

// GetInstallOrder returns the plugins in the order they should be installed
// using topological sort to resolve dependencies
func (dg *DependencyGraph) GetInstallOrder(targetPlugins []string) ([]string, error) {
	if len(targetPlugins) == 0 {
		return []string{}, nil
	}
	
	// Get all required plugins including dependencies
	required := make(map[string]bool)
	for _, plugin := range targetPlugins {
		if err := dg.collectDependencies(plugin, required); err != nil {
			return nil, err
		}
	}
	
	// Convert to slice for topological sort
	plugins := make([]string, 0, len(required))
	for plugin := range required {
		plugins = append(plugins, plugin)
	}
	
	return dg.topologicalSort(plugins)
}

// GetUninstallOrder returns the plugins in the order they should be uninstalled
// (reverse of install order, considering dependents)
func (dg *DependencyGraph) GetUninstallOrder(targetPlugins []string) ([]string, error) {
	if len(targetPlugins) == 0 {
		return []string{}, nil
	}
	
	// Get all plugins that need to be uninstalled including dependents
	toUninstall := make(map[string]bool)
	for _, plugin := range targetPlugins {
		if err := dg.collectDependents(plugin, toUninstall); err != nil {
			return nil, err
		}
	}
	
	// Convert to slice for topological sort
	plugins := make([]string, 0, len(toUninstall))
	for plugin := range toUninstall {
		plugins = append(plugins, plugin)
	}
	
	// Get install order and reverse it for uninstall
	order, err := dg.topologicalSort(plugins)
	if err != nil {
		return nil, err
	}
	
	// Reverse the order for uninstallation
	for i, j := 0, len(order)-1; i < j; i, j = i+1, j-1 {
		order[i], order[j] = order[j], order[i]
	}
	
	return order, nil
}

// ValidateInstall validates that dependencies are met for installation
func (dg *DependencyGraph) ValidateInstall(pluginName string, installedPlugins []string) error {
	node := dg.nodes[pluginName]
	if node == nil || node.Plugin == nil {
		return fmt.Errorf("plugin '%s' not found in dependency graph", pluginName)
	}
	
	installedSet := make(map[string]bool)
	for _, p := range installedPlugins {
		installedSet[p] = true
	}
	
	missingDeps := make([]string, 0)
	for _, dep := range node.Dependencies {
		if !installedSet[dep] {
			missingDeps = append(missingDeps, dep)
		}
	}
	
	if len(missingDeps) > 0 {
		return fmt.Errorf("plugin '%s' has unmet dependencies: %s", 
			pluginName, strings.Join(missingDeps, ", "))
	}
	
	return nil
}

// ValidateUninstall validates that no installed plugins depend on the plugin being uninstalled
func (dg *DependencyGraph) ValidateUninstall(pluginName string, installedPlugins []string) error {
	installedSet := make(map[string]bool)
	for _, p := range installedPlugins {
		installedSet[p] = true
	}
	
	node := dg.nodes[pluginName]
	if node == nil {
		return nil // Plugin not in graph, safe to uninstall
	}
	
	blockers := make([]string, 0)
	for _, dependent := range node.Dependents {
		if installedSet[dependent] {
			blockers = append(blockers, dependent)
		}
	}
	
	if len(blockers) > 0 {
		return fmt.Errorf("cannot uninstall '%s': the following installed plugins depend on it: %s", 
			pluginName, strings.Join(blockers, ", "))
	}
	
	return nil
}

// GetDependencies returns the direct dependencies of a plugin
func (dg *DependencyGraph) GetDependencies(pluginName string) []string {
	node := dg.nodes[pluginName]
	if node == nil {
		return []string{}
	}
	return node.Dependencies
}

// GetDependents returns the plugins that depend on the given plugin
func (dg *DependencyGraph) GetDependents(pluginName string) []string {
	node := dg.nodes[pluginName]
	if node == nil {
		return []string{}
	}
	return node.Dependents
}

// HasCycles detects if there are circular dependencies in the graph
func (dg *DependencyGraph) HasCycles() bool {
	// Use DFS to detect cycles
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	
	for pluginName := range dg.nodes {
		if !visited[pluginName] {
			if dg.hasCyclesDFS(pluginName, visited, recStack) {
				return true
			}
		}
	}
	
	return false
}

// collectDependencies recursively collects all dependencies for a plugin
func (dg *DependencyGraph) collectDependencies(pluginName string, collected map[string]bool) error {
	return dg.collectDependenciesWithStack(pluginName, collected, make(map[string]bool))
}

// collectDependenciesWithStack recursively collects dependencies with cycle detection
func (dg *DependencyGraph) collectDependenciesWithStack(pluginName string, collected, stack map[string]bool) error {
	if collected[pluginName] {
		return nil // Already processed
	}
	
	if stack[pluginName] {
		return fmt.Errorf("circular dependency detected involving plugin '%s'", pluginName)
	}
	
	node := dg.nodes[pluginName]
	if node == nil || node.Plugin == nil {
		return fmt.Errorf("plugin '%s' not found", pluginName)
	}
	
	// Mark as being processed
	stack[pluginName] = true
	
	// First, collect dependencies
	for _, dep := range node.Dependencies {
		if err := dg.collectDependenciesWithStack(dep, collected, stack); err != nil {
			return err
		}
	}
	
	// Remove from processing stack and mark as collected
	stack[pluginName] = false
	collected[pluginName] = true
	return nil
}

// collectDependents recursively collects all dependents for a plugin
func (dg *DependencyGraph) collectDependents(pluginName string, collected map[string]bool) error {
	if collected[pluginName] {
		return nil // Already processed
	}
	
	// Add the plugin itself first
	collected[pluginName] = true
	
	node := dg.nodes[pluginName]
	if node == nil {
		return nil // Plugin not in graph
	}
	
	// Then collect all dependents
	for _, dependent := range node.Dependents {
		if err := dg.collectDependents(dependent, collected); err != nil {
			return err
		}
	}
	
	return nil
}

// topologicalSort performs topological sort using Kahn's algorithm
func (dg *DependencyGraph) topologicalSort(plugins []string) ([]string, error) {
	// Calculate in-degree for each plugin
	inDegree := make(map[string]int)
	for _, plugin := range plugins {
		inDegree[plugin] = 0
	}
	
	for _, plugin := range plugins {
		node := dg.nodes[plugin]
		if node != nil {
			for _, dep := range node.Dependencies {
				if _, exists := inDegree[dep]; exists {
					inDegree[plugin]++
				}
			}
		}
	}
	
	// Initialize queue with plugins having no dependencies
	queue := make([]string, 0)
	for plugin, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, plugin)
		}
	}
	
	result := make([]string, 0, len(plugins))
	
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)
		
		// Update in-degree for dependents
		for _, dependent := range plugins {
			node := dg.nodes[dependent]
			if node != nil {
				for _, dep := range node.Dependencies {
					if dep == current {
						inDegree[dependent]--
						if inDegree[dependent] == 0 {
							queue = append(queue, dependent)
						}
					}
				}
			}
		}
	}
	
	if len(result) != len(plugins) {
		return nil, fmt.Errorf("circular dependency detected")
	}
	
	return result, nil
}

// hasCyclesDFS detects cycles using depth-first search
func (dg *DependencyGraph) hasCyclesDFS(plugin string, visited, recStack map[string]bool) bool {
	visited[plugin] = true
	recStack[plugin] = true
	
	node := dg.nodes[plugin]
	if node != nil {
		for _, dep := range node.Dependencies {
			if !visited[dep] {
				if dg.hasCyclesDFS(dep, visited, recStack) {
					return true
				}
			} else if recStack[dep] {
				return true
			}
		}
	}
	
	recStack[plugin] = false
	return false
}

// DependencyValidator provides high-level validation methods
type DependencyValidator struct {
	graph *DependencyGraph
}

// NewDependencyValidator creates a new dependency validator
func NewDependencyValidator(plugins []DependencyPlugin) *DependencyValidator {
	graph := NewDependencyGraph()
	
	for _, plugin := range plugins {
		graph.AddPlugin(plugin)
	}
	
	if graph.HasCycles() {
		logger.Error("Circular dependency detected in plugin graph")
	}
	
	return &DependencyValidator{
		graph: graph,
	}
}

// ValidateInstallation validates installation of multiple plugins
func (dv *DependencyValidator) ValidateInstallation(targetPlugins []string, installedPlugins []string) ([]string, error) {
	logger.Infoln("Validating plugin installation dependencies...")
	
	installOrder, err := dv.graph.GetInstallOrder(targetPlugins)
	if err != nil {
		return nil, fmt.Errorf("failed to determine install order: %w", err)
	}
	
	installedSet := make(map[string]bool)
	for _, p := range installedPlugins {
		installedSet[p] = true
	}
	
	// Filter out already installed plugins and validate remaining ones
	needsInstallation := make([]string, 0)
	for _, plugin := range installOrder {
		if !installedSet[plugin] {
			if err := dv.graph.ValidateInstall(plugin, installedPlugins); err != nil {
				return nil, err
			}
			needsInstallation = append(needsInstallation, plugin)
			// Add to installed set for next validation
			installedPlugins = append(installedPlugins, plugin)
			installedSet[plugin] = true
		}
	}
	
	logger.Successln("Dependency validation passed")
	return needsInstallation, nil
}

// ValidateUninstallation validates uninstallation of multiple plugins
func (dv *DependencyValidator) ValidateUninstallation(targetPlugins []string, installedPlugins []string) ([]string, error) {
	logger.Infoln("Validating plugin uninstallation dependencies...")
	
	uninstallOrder, err := dv.graph.GetUninstallOrder(targetPlugins)
	if err != nil {
		return nil, fmt.Errorf("failed to determine uninstall order: %w", err)
	}
	
	installedSet := make(map[string]bool)
	for _, p := range installedPlugins {
		installedSet[p] = true
	}
	
	// Filter out not-installed plugins and validate remaining ones
	needsUninstallation := make([]string, 0)
	for _, plugin := range uninstallOrder {
		if installedSet[plugin] {
			if err := dv.graph.ValidateUninstall(plugin, installedPlugins); err != nil {
				return nil, err
			}
			needsUninstallation = append(needsUninstallation, plugin)
			// Remove from installed set for next validation
			newInstalled := make([]string, 0)
			for _, p := range installedPlugins {
				if p != plugin {
					newInstalled = append(newInstalled, p)
				}
			}
			installedPlugins = newInstalled
			installedSet[plugin] = false
		}
	}
	
	logger.Successln("Dependency validation passed")
	return needsUninstallation, nil
}

// GetDependencyInfo returns dependency information for a plugin
func (dv *DependencyValidator) GetDependencyInfo(pluginName string) (dependencies []string, dependents []string) {
	return dv.graph.GetDependencies(pluginName), dv.graph.GetDependents(pluginName)
} 