package plugins

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewObservability(t *testing.T) {
	kubeConfig := "/tmp/test-kubeconfig"
	
	obs, err := NewObservability(kubeConfig)
	
	assert.NoError(t, err)
	assert.NotNil(t, obs)
	assert.Equal(t, kubeConfig, obs.KubeConfig)
	assert.NotNil(t, obs.BasePlugin)
}

func TestObservability_GetName(t *testing.T) {
	obs, err := NewObservability("/tmp/test-kubeconfig")
	assert.NoError(t, err)
	
	name := obs.GetName()
	assert.Equal(t, "observability", name)
}

func TestObservability_GetOptions(t *testing.T) {
	obs, err := NewObservability("/tmp/test-kubeconfig")
	assert.NoError(t, err)
	
	options := obs.GetOptions()
	
	assert.NotNil(t, options.Version)
	assert.Equal(t, ObservabilityChartVersion, *options.Version)
	assert.NotNil(t, options.Namespace)
	assert.Equal(t, ObservabilityNamespace, *options.Namespace)
	assert.NotNil(t, options.ChartName)
	assert.Equal(t, ObservabilityChartName, *options.ChartName)
	assert.NotNil(t, options.RepoName)
	assert.Equal(t, ObservabilityRepoName, *options.RepoName)
	assert.NotNil(t, options.Repository)
	assert.Equal(t, ObservabilityRepoURL, *options.Repository)
	assert.Equal(t, "operator.victoriametrics.com", options.CRDsGroupVersion)
	assert.NotNil(t, options.ChartValues)
}

func TestObservability_GetDependencies(t *testing.T) {
	obs, err := NewObservability("/tmp/test-kubeconfig")
	assert.NoError(t, err)
	
	deps := obs.GetDependencies()
	assert.Empty(t, deps, "Observability plugin should have no dependencies")
}

func TestObservability_ChartValues(t *testing.T) {
	obs, err := NewObservability("/tmp/test-kubeconfig")
	assert.NoError(t, err)
	
	values := obs.getChartValues()
	
	// Check that essential components are enabled
	assert.NotNil(t, values["vmsingle"])
	vmSingle := values["vmsingle"].(map[string]interface{})
	assert.True(t, vmSingle["enabled"].(bool))
	
	assert.NotNil(t, values["grafana"])
	grafana := values["grafana"].(map[string]interface{})
	assert.True(t, grafana["enabled"].(bool))
	
	assert.NotNil(t, values["nodeExporter"])
	nodeExporter := values["nodeExporter"].(map[string]interface{})
	assert.True(t, nodeExporter["enabled"].(bool))
	
	assert.NotNil(t, values["kubeStateMetrics"])
	kubeStateMetrics := values["kubeStateMetrics"].(map[string]interface{})
	assert.True(t, kubeStateMetrics["enabled"].(bool))
	
	assert.NotNil(t, values["vlogs"])
	vlogs := values["vlogs"].(map[string]interface{})
	assert.True(t, vlogs["enabled"].(bool))
	
	assert.NotNil(t, values["alertmanager"])
	alertmanager := values["alertmanager"].(map[string]interface{})
	assert.True(t, alertmanager["enabled"].(bool))
	
	assert.NotNil(t, values["jaeger"])
	jaeger := values["jaeger"].(map[string]interface{})
	assert.True(t, jaeger["enabled"].(bool))
} 