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
		"admin.password":                           true,
		"server.replicas":                          true,
		"server.resources.requests.memory":         true,
		"server.resources.requests.cpu":            true,
		"server.resources.limits.memory":           true,
		"server.resources.limits.cpu":              true,
		"redis.enabled":                            true,
		"redis.resources.requests.memory":          true,
		"redis.resources.requests.cpu":             true,
		"applicationSet.enabled":                   true,
		"notifications.enabled":                    true,
		"dex.enabled":                              true,
		"server.service.type":                      true,
		"server.ingress.enabled":                   true,
		"configs.secret.argocdServerAdminPassword": true,
	}

	for key := range overrides {
		if !allowedKeys[key] {
			return fmt.Errorf("override key '%s' is not allowed for argocd plugin. Allowed keys: %v", key, getKeys(allowedKeys))
		}
	}

	return nil
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
	// Get default values
	defaultValues, err := a.getValuesContent()
	if err != nil {
		logger.Errorln("failed to get values content: %v", err)
		return a.overrideValues // Return only override values if default fetch fails
	}

	// If no override values, return defaults
	if len(a.overrideValues) == 0 {
		return defaultValues
	}

	// Merge override values with defaults
	mergedValues := mergeValues(defaultValues, a.overrideValues)
	logger.Debugf("ArgoCD values merged with overrides: %v", a.overrideValues)

	return mergedValues
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
