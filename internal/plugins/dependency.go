package plugins

import (
	"fmt"
	"strings"

	"github.com/mrgb7/playground/pkg/logger"
)

type DependencyPlugin interface {
	Plugin
	GetDependencies() []string
}

type GraphNode struct {
	Plugin       DependencyPlugin
	Dependencies []string // plugins this node depends on
	Dependents   []string // plugins that depend on this node
}

type DependencyGraph struct {
	nodes map[string]*GraphNode
}

func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes: make(map[string]*GraphNode),
	}
}

func (dg *DependencyGraph) AddPlugin(plugin DependencyPlugin) {
	name := plugin.GetName()
	dependencies := plugin.GetDependencies()

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

	for _, dep := range dependencies {
		if dg.nodes[dep] == nil {
			dg.nodes[dep] = &GraphNode{
				Plugin:       nil,
				Dependencies: make([]string, 0),
				Dependents:   make([]string, 0),
			}
		}

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

func (dg *DependencyGraph) GetInstallOrder(targetPlugins []string) ([]string, error) {
	if len(targetPlugins) == 0 {
		return []string{}, nil
	}

	required := make(map[string]bool)
	for _, plugin := range targetPlugins {
		if err := dg.collectDependencies(plugin, required); err != nil {
			return nil, err
		}
	}

	plugins := make([]string, 0, len(required))
	for plugin := range required {
		plugins = append(plugins, plugin)
	}

	return dg.topologicalSort(plugins)
}

func (dg *DependencyGraph) GetUninstallOrder(targetPlugins []string) ([]string, error) {
	if len(targetPlugins) == 0 {
		return []string{}, nil
	}

	toUninstall := make(map[string]bool)
	for _, plugin := range targetPlugins {
		if err := dg.collectDependents(plugin, toUninstall); err != nil {
			return nil, err
		}
	}

	plugins := make([]string, 0, len(toUninstall))
	for plugin := range toUninstall {
		plugins = append(plugins, plugin)
	}

	order, err := dg.topologicalSort(plugins)
	if err != nil {
		return nil, err
	}

	for i, j := 0, len(order)-1; i < j; i, j = i+1, j-1 {
		order[i], order[j] = order[j], order[i]
	}

	return order, nil
}

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

func (dg *DependencyGraph) ValidateUninstall(pluginName string, installedPlugins []string) error {
	installedSet := make(map[string]bool)
	for _, p := range installedPlugins {
		installedSet[p] = true
	}

	node := dg.nodes[pluginName]
	if node == nil {
		return nil
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

func (dg *DependencyGraph) GetDependencies(pluginName string) []string {
	node := dg.nodes[pluginName]
	if node == nil {
		return []string{}
	}
	return node.Dependencies
}

func (dg *DependencyGraph) GetDependents(pluginName string) []string {
	node := dg.nodes[pluginName]
	if node == nil {
		return []string{}
	}
	return node.Dependents
}

func (dg *DependencyGraph) HasCycles() bool {
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

func (dg *DependencyGraph) collectDependencies(pluginName string, collected map[string]bool) error {
	return dg.collectDependenciesWithStack(pluginName, collected, make(map[string]bool))
}

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

	stack[pluginName] = true

	for _, dep := range node.Dependencies {
		if err := dg.collectDependenciesWithStack(dep, collected, stack); err != nil {
			return err
		}
	}

	stack[pluginName] = false
	collected[pluginName] = true
	return nil
}

// collectDependents recursively collects all dependents for a plugin
func (dg *DependencyGraph) collectDependents(pluginName string, collected map[string]bool) error {
	if collected[pluginName] {
		return nil // Already processed
	}

	collected[pluginName] = true

	node := dg.nodes[pluginName]
	if node == nil {
		return nil // Plugin not in graph
	}

	for _, dependent := range node.Dependents {
		if err := dg.collectDependents(dependent, collected); err != nil {
			return err
		}
	}

	return nil
}

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

type DependencyValidator struct {
	graph *DependencyGraph
}

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

func (dv *DependencyValidator) GetDependencyInfo(pluginName string) (dependencies []string, dependents []string) {
	return dv.graph.GetDependencies(pluginName), dv.graph.GetDependents(pluginName)
}
