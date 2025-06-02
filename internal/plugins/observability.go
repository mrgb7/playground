package plugins

import (
	"context"
	"time"

	"github.com/mrgb7/playground/internal/k8s"
	"github.com/mrgb7/playground/pkg/logger"
)

type Observability struct {
	KubeConfig string
	*BasePlugin
}

var (
	ObservabilityRepoURL      = "https://victoria-metrics.github.io/helm-charts/"
	ObservabilityChartName    = "victoria-metrics-k8s-stack"
	ObservabilityChartVersion = "0.27.3"
	ObservabilityReleaseName  = "observability"
	ObservabilityNamespace    = "monitoring"
	ObservabilityRepoName     = "victoria-metrics"
)

func NewObservability(kubeConfig string) (*Observability, error) {
	obs := &Observability{
		KubeConfig: kubeConfig,
	}
	obs.BasePlugin = NewBasePlugin(kubeConfig, obs)
	return obs, nil
}

func (o *Observability) GetName() string {
	return "observability"
}

func (o *Observability) GetOptions() PluginOptions {
	return PluginOptions{
		Version:          &ObservabilityChartVersion,
		Namespace:        &ObservabilityNamespace,
		ChartName:        &ObservabilityChartName,
		RepoName:         &ObservabilityRepoName,
		Repository:       &ObservabilityRepoURL,
		releaseName:      &ObservabilityReleaseName,
		ChartValues:      o.getChartValues(),
		CRDsGroupVersion: "operator.victoriametrics.com",
	}
}

func (o *Observability) Install(kubeConfig, clusterName string, ensure ...bool) error {
	return o.UnifiedInstall(kubeConfig, clusterName, ensure...)
}

func (o *Observability) Uninstall(kubeConfig, clusterName string, ensure ...bool) error {
	return o.UnifiedUninstall(kubeConfig, clusterName, ensure...)
}

func (o *Observability) Status() string {
	c, err := k8s.NewK8sClient(o.KubeConfig)
	if err != nil {
		logger.Debugf("failed to create k8s client: %v", err)
		return StatusUnknown
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ns, err := c.GetNameSpace(ObservabilityNamespace, ctx)
	if ns == "" || err != nil {
		logger.Debugf("failed to get observability namespace: %v", err)
		return StatusNotInstalled
	}
	return StatusRunning
}

func (o *Observability) GetDependencies() []string {
	return []string{}
}

func (o *Observability) getChartValues() map[string]interface{} {
	return map[string]interface{}{
		// Victoria Metrics configuration
		"vmsingle": map[string]interface{}{
			"enabled": true,
			"spec": map[string]interface{}{
				"retentionPeriod": "30d",
				"storage": map[string]interface{}{
					"storageClassName": "local-path",
					"accessModes":      []string{"ReadWriteOnce"},
					"size":             "20Gi",
				},
			},
		},
		// Node Exporter configuration
		"nodeExporter": map[string]interface{}{
			"enabled": true,
		},
		// Kube State Metrics configuration
		"kubeStateMetrics": map[string]interface{}{
			"enabled": true,
		},
		// Grafana configuration
		"grafana": map[string]interface{}{
			"enabled": true,
			"spec": map[string]interface{}{
				"storage": map[string]interface{}{
					"storageClassName": "local-path",
					"accessModes":      []string{"ReadWriteOnce"},
					"size":             "5Gi",
				},
			},
			"sidecar": map[string]interface{}{
				"datasources": map[string]interface{}{
					"enabled": true,
				},
				"dashboards": map[string]interface{}{
					"enabled": true,
				},
			},
		},
		// Victoria Logs configuration
		"vlogs": map[string]interface{}{
			"enabled": true,
			"spec": map[string]interface{}{
				"storage": map[string]interface{}{
					"storageClassName": "local-path",
					"accessModes":      []string{"ReadWriteOnce"},
					"size":             "10Gi",
				},
			},
		},
		// Fluent Bit configuration for log collection
		"fluent-bit": map[string]interface{}{
			"enabled": true,
		},
		// Alert Manager configuration
		"alertmanager": map[string]interface{}{
			"enabled": true,
			"spec": map[string]interface{}{
				"storage": map[string]interface{}{
					"storageClassName": "local-path",  
					"accessModes":      []string{"ReadWriteOnce"},
					"size":             "2Gi",
				},
			},
		},
		// Victoria Metrics Alert configuration
		"vmalert": map[string]interface{}{
			"enabled": true,
		},
		// Jaeger configuration for tracing
		"jaeger": map[string]interface{}{
			"enabled": true,
			"spec": map[string]interface{}{
				"storage": map[string]interface{}{
					"type": "memory",
				},
			},
		},
		// OpenTelemetry Collector configuration
		"opentelemetry-collector": map[string]interface{}{
			"enabled": true,
		},
		// Default service monitors
		"defaultRules": map[string]interface{}{
			"enabled": true,
		},
		// Prometheus compatibility
		"prometheus": map[string]interface{}{
			"enabled": false, // Using Victoria Metrics instead
		},
	}
} 