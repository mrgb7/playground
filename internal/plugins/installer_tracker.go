package plugins

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mrgb7/playground/internal/k8s"
	"github.com/mrgb7/playground/pkg/logger"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// InstallerTrackerConfigMapName is the name of the ConfigMap used to track plugin installers
	InstallerTrackerConfigMapName = "playground-plugin-installer-tracker"
	// InstallerTrackerNamespace is the namespace where the tracker ConfigMap is stored
	InstallerTrackerNamespace = "kube-system"
	// InstallerTypeHelm represents Helm installer
	InstallerTypeHelm = "helm"
	// InstallerTypeArgoCD represents ArgoCD installer
	InstallerTypeArgoCD = "argocd"
)

// InstallerTracker manages tracking of which installer was used for each plugin
type InstallerTracker struct {
	kubeConfig string
	k8sClient  *k8s.K8sClient
}

// NewInstallerTracker creates a new installer tracker
func NewInstallerTracker(kubeConfig string) (*InstallerTracker, error) {
	client, err := k8s.NewK8sClient(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	return &InstallerTracker{
		kubeConfig: kubeConfig,
		k8sClient:  client,
	}, nil
}

// RecordPluginInstaller records which installer was used for a plugin
func (t *InstallerTracker) RecordPluginInstaller(pluginName, installerType string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	configMap, err := t.getOrCreateTrackerConfigMap(ctx)
	if err != nil {
		return fmt.Errorf("failed to get or create tracker ConfigMap: %w", err)
	}

	if configMap.Data == nil {
		configMap.Data = make(map[string]string)
	}

	configMap.Data[pluginName] = installerType

	_, err = t.k8sClient.Clientset.CoreV1().ConfigMaps(InstallerTrackerNamespace).Update(ctx, configMap, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update tracker ConfigMap: %w", err)
	}

	logger.Debugln("Recorded installer type '%s' for plugin '%s'", installerType, pluginName)
	return nil
}

// GetPluginInstaller retrieves which installer was used for a plugin
func (t *InstallerTracker) GetPluginInstaller(pluginName string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	configMap, err := t.k8sClient.Clientset.CoreV1().ConfigMaps(InstallerTrackerNamespace).Get(
		ctx, InstallerTrackerConfigMapName, metav1.GetOptions{})
	if err != nil {
		// If ConfigMap doesn't exist, return empty (no tracking info)
		if strings.Contains(err.Error(), "not found") {
			logger.Debugln("Tracker ConfigMap not found, no installer recorded for plugin '%s'", pluginName)
			return "", nil
		}
		return "", fmt.Errorf("failed to get tracker ConfigMap: %w", err)
	}

	if configMap.Data == nil {
		return "", nil
	}

	installerType := configMap.Data[pluginName]
	logger.Debugln("Found recorded installer type '%s' for plugin '%s'", installerType, pluginName)
	return installerType, nil
}

// RemovePluginInstaller removes the tracking record for a plugin
func (t *InstallerTracker) RemovePluginInstaller(pluginName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	configMap, err := t.k8sClient.Clientset.CoreV1().ConfigMaps(InstallerTrackerNamespace).Get(
		ctx, InstallerTrackerConfigMapName, metav1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			logger.Debugln("Tracker ConfigMap not found, nothing to remove for plugin '%s'", pluginName)
			return nil
		}
		return fmt.Errorf("failed to get tracker ConfigMap: %w", err)
	}

	if configMap.Data == nil {
		return nil
	}

	delete(configMap.Data, pluginName)

	_, err = t.k8sClient.Clientset.CoreV1().ConfigMaps(InstallerTrackerNamespace).Update(ctx, configMap, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update tracker ConfigMap: %w", err)
	}

	logger.Debugln("Removed installer tracking record for plugin '%s'", pluginName)
	return nil
}

// getOrCreateTrackerConfigMap gets the tracker ConfigMap or creates it if it doesn't exist
func (t *InstallerTracker) getOrCreateTrackerConfigMap(ctx context.Context) (*v1.ConfigMap, error) {
	configMap, err := t.k8sClient.Clientset.CoreV1().ConfigMaps(InstallerTrackerNamespace).Get(
		ctx, InstallerTrackerConfigMapName, metav1.GetOptions{})
	if err == nil {
		return configMap, nil
	}

	if !strings.Contains(err.Error(), "not found") {
		return nil, fmt.Errorf("failed to get tracker ConfigMap: %w", err)
	}

	// ConfigMap doesn't exist, create it
	newConfigMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      InstallerTrackerConfigMapName,
			Namespace: InstallerTrackerNamespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "playground",
				"app.kubernetes.io/component":  "installer-tracker",
				"app.kubernetes.io/managed-by": "playground",
			},
		},
		Data: make(map[string]string),
	}

	createdConfigMap, err := t.k8sClient.Clientset.CoreV1().ConfigMaps(InstallerTrackerNamespace).Create(
		ctx, newConfigMap, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create tracker ConfigMap: %w", err)
	}

	logger.Debugln("Created new installer tracker ConfigMap")
	return createdConfigMap, nil
}
