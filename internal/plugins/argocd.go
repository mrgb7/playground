package plugins

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mrgb7/playground/internal/installer"
	"github.com/mrgb7/playground/internal/k8s"
	"github.com/mrgb7/playground/pkg/logger"
	"gopkg.in/yaml.v3"
)

type Argocd struct {
	KubeConfig string
	*BasePlugin
	Tracker        *InstallerTracker
	overrideValues map[string]interface{}
}

var (
	ArgocdRepoURL       = "https://argoproj.github.io/argo-helm"
	ArgocdChartName     = "argo-cd"
	ArgocdChartVersion  = "8.0.0"
	ArgocdReleaseName   = "argocd"
	ArgocdNamespace     = "argocd"
	ArgoRepoName        = "argo"
	ArgocdValuesFileURL = "https://raw.githubusercontent.com/mrgb7/core-infrastructure/" +
		"refs/heads/main/argocd/argocd-values-local.yaml"
)

const (
	HTTPTimeoutSeconds = 30
	MaxResponseSize    = 10 * 1024 * 1024
)

func NewArgocd(kubeConfig string) (*Argocd, error) {
	t, err := NewInstallerTracker(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create installer tracker: %w", err)
	}
	argo := &Argocd{
		KubeConfig:     kubeConfig,
		Tracker:        t,
		overrideValues: make(map[string]interface{}),
	}
	argo.BasePlugin = NewBasePlugin(kubeConfig, argo)
	return argo, nil
}

func (a *Argocd) GetName() string {
	return "argocd"
}

func (a *Argocd) GetOptions() PluginOptions {
	return PluginOptions{
		Version:          &ArgocdChartVersion,
		Namespace:        &ArgocdNamespace,
		ChartName:        &ArgocdChartName,
		RepoName:         &ArgoRepoName,
		Repository:       &ArgocdRepoURL,
		releaseName:      &ArgocdReleaseName,
		ChartValues:      a.getChartValues(),
		CRDsGroupVersion: "argoproj.io",
	}
}

func (a *Argocd) Install(kubeConfig, clusterName string, ensure ...bool) error {
	return a.UnifiedInstall(kubeConfig, clusterName, ensure...)
}

func (a *Argocd) Uninstall(kubeConfig, clusterName string, ensure ...bool) error {
	if err := a.checkUsage(); err != nil {
		return err
	}

	return a.UnifiedUninstall(kubeConfig, clusterName, ensure...)
}

// ValidateOverrideValues implements OverrideValidator interface
func (a *Argocd) ValidateOverrideValues(overrides map[string]interface{}) error {
	allowedKeys := map[string]bool{
		"admin.password": true,
	}

	// Flatten the nested override structure back to dot notation for validation
	flattenedKeys := flattenKeys(overrides, "")

	for _, key := range flattenedKeys {
		if !allowedKeys[key] {
			return fmt.Errorf("override key '%s' is not allowed for argocd plugin. Allowed keys: %v", key, getKeys(allowedKeys))
		}
	}

	return nil
}

// flattenKeys converts nested map structure back to dot notation keys for validation
func flattenKeys(m map[string]interface{}, prefix string) []string {
	var keys []string

	for k, v := range m {
		var fullKey string
		if prefix == "" {
			fullKey = k
		} else {
			fullKey = prefix + "." + k
		}

		if nestedMap, ok := v.(map[string]interface{}); ok {
			// Recursively flatten nested maps
			keys = append(keys, flattenKeys(nestedMap, fullKey)...)
		} else {
			// This is a leaf value, add the full key
			keys = append(keys, fullKey)
		}
	}

	return keys
}

// SetOverrideValues implements OverridablePlugin interface
func (a *Argocd) SetOverrideValues(overrides map[string]interface{}) {
	a.overrideValues = overrides
}

func getKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func (a *Argocd) checkUsage() error {
	plugins, _ := a.Tracker.GetAllPluginByInstaller(a.GetName())

	if len(plugins) > 0 {
		return fmt.Errorf("you cannot uninstall argocd because it is used by other plugins: %v", plugins)
	}
	return nil
}

func (a *Argocd) getValuesContent() (map[string]interface{}, error) {
	if _, err := url.Parse(ArgocdValuesFileURL); err != nil {
		return nil, fmt.Errorf("invalid values file URL: %w", err)
	}

	httpClient := &http.Client{
		Timeout: HTTPTimeoutSeconds * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), HTTPTimeoutSeconds*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", ArgocdValuesFileURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch values file: %w", err)
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			logger.Debugln("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch values file: HTTP %d %s", resp.StatusCode, resp.Status)
	}

	limitedReader := io.LimitReader(resp.Body, MaxResponseSize)
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	hash := sha256.Sum256(content)
	logger.Debugf("ArgoCD values file SHA256: %x", hash)

	var values map[string]interface{}
	if err := yaml.Unmarshal(content, &values); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML content: %w", err)
	}

	return values, nil
}

func (a *Argocd) Status() string {
	c, err := k8s.NewK8sClient(a.KubeConfig)
	if err != nil {
		logger.Debugf("failed to create k8s client: %v", err)
		return StatusUnknown
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ns, err := c.GetNameSpace(ArgocdNamespace, ctx)
	if ns == "" || err != nil {
		logger.Debugf("failed to get argocd namespace: %v", err)
		return StatusNotInstalled
	}
	return StatusRunning
}

func (a *Argocd) getChartValues() map[string]interface{} {
	// Get default values from remote
	defaultValues, err := a.getValuesContent()
	if err != nil {
		logger.Errorln("failed to get default values content: %v", err)
		defaultValues = make(map[string]interface{})
	}

	// If no override values, return defaults (normal flow)
	if len(a.overrideValues) == 0 {
		return defaultValues
	}

	// For override mode, perform three-way merge:
	// 1. Default values (base)
	// 2. Current installed values (includes modifications by other plugins)
	// 3. User override values (highest priority)

	currentValues := a.getCurrentInstalledValues()

	// Three-way merge: defaults -> current -> overrides
	mergedValues := mergeValues(defaultValues, currentValues)
	finalValues := mergeValues(mergedValues, a.overrideValues)

	logger.Debugf("ArgoCD three-way merge - defaults: %d keys, current: %d keys, overrides: %d keys",
		len(defaultValues), len(currentValues), len(a.overrideValues))

	return finalValues
}

// getCurrentInstalledValues retrieves the current Helm values for the installed ArgoCD instance
func (a *Argocd) getCurrentInstalledValues() map[string]interface{} {
	// Try to get current values from Helm release
	currentValues, err := a.getHelmReleaseValues()
	if err != nil {
		logger.Warnf("Failed to get current installed values for ArgoCD: %v", err)
		logger.Debugf("Will proceed with defaults + overrides only")
		return make(map[string]interface{})
	}

	logger.Debugf("Retrieved current ArgoCD values with %d top-level keys", len(currentValues))
	return currentValues
}

// getHelmReleaseValues retrieves the current values from the Helm release
func (a *Argocd) getHelmReleaseValues() (map[string]interface{}, error) {
	// Check what installer type was used for this plugin
	installerType, err := a.Tracker.GetPluginInstaller(a.GetName())
	if err != nil {
		return nil, fmt.Errorf("failed to get installer type: %w", err)
	}

	switch installerType {
	case InstallerTypeHelm:
		return a.getHelmValues()
	case InstallerTypeArgoCD:
		return a.getArgoCDApplicationValues()
	default:
		logger.Warnf("Unknown installer type '%s' for ArgoCD, cannot retrieve current values", installerType)
		return make(map[string]interface{}), nil
	}
}

// getHelmValues retrieves values from Helm release
func (a *Argocd) getHelmValues() (map[string]interface{}, error) {
	// Create a Helm installer to access Helm functionality
	helmInstaller, err := installer.NewHelmInstaller(a.KubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Helm installer: %w", err)
	}

	// Use the Helm installer to get current values
	currentValues, err := helmInstaller.GetCurrentValues(ArgocdReleaseName, ArgocdNamespace)
	if err != nil {
		// If release doesn't exist or other error, return empty values
		logger.Debugf("Could not retrieve Helm values for %s: %v", ArgocdReleaseName, err)
		return make(map[string]interface{}), nil
	}

	return currentValues, nil
}

// getArgoCDApplicationValues retrieves values from ArgoCD Application resource
func (a *Argocd) getArgoCDApplicationValues() (map[string]interface{}, error) {
	// For now, return empty map - this would need K8s client to get Application resource
	// TODO: Implement actual ArgoCD Application values retrieval
	logger.Debugf("ArgoCD Application values retrieval not yet implemented, using empty values")
	return make(map[string]interface{}), nil
}

// mergeValues deeply merges override values into default values
func mergeValues(defaults, overrides map[string]interface{}) map[string]interface{} {
	if defaults == nil {
		defaults = make(map[string]interface{})
	}

	result := make(map[string]interface{})

	// Copy defaults
	for k, v := range defaults {
		result[k] = v
	}

	// Apply overrides with proper deep merging
	for key, value := range overrides {
		if strings.Contains(key, ".") {
			// Handle dot notation keys
			setNestedMapValue(result, key, value)
		} else {
			// Handle direct keys with potential deep merging
			if existingValue, exists := result[key]; exists {
				if existingMap, ok := existingValue.(map[string]interface{}); ok {
					if valueMap, ok := value.(map[string]interface{}); ok {
						// Both are maps, merge them recursively
						result[key] = mergeValues(existingMap, valueMap)
					} else {
						// Override with the new value
						result[key] = value
					}
				} else {
					// Override with the new value
					result[key] = value
				}
			} else {
				// Key doesn't exist, add it
				result[key] = value
			}
		}
	}

	return result
}

// setNestedMapValue sets a value in a nested map using dot notation
func setNestedMapValue(m map[string]interface{}, key string, value interface{}) {
	keys := splitKey(key)
	current := m

	// Navigate to the nested map, creating maps as needed
	for _, k := range keys[:len(keys)-1] {
		if _, exists := current[k]; !exists {
			current[k] = make(map[string]interface{})
		}
		if nested, ok := current[k].(map[string]interface{}); ok {
			current = nested
		} else {
			// If key exists but is not a map, replace it
			current[k] = make(map[string]interface{})
			current = current[k].(map[string]interface{})
		}
	}

	// Set the final value
	finalKey := keys[len(keys)-1]
	current[finalKey] = value
}

// splitKey splits a dot-notation key into parts
func splitKey(key string) []string {
	return strings.Split(key, ".")
}

func (a *Argocd) GetDependencies() []string {
	return []string{}
}
