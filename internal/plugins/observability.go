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
	LightweightMode bool // Support for lightweight vs full stack
}

var (
	ObservabilityRepoURL      = "https://victoriametrics.github.io/helm-charts/"
	ObservabilityChartName    = "victoria-metrics-k8s-stack"
	ObservabilityChartVersion = "0.50.1"
	ObservabilityReleaseName  = "obs"
	ObservabilityNamespace    = "monitoring"
	ObservabilityRepoName     = "victoria-metrics"
)

func NewObservability(kubeConfig string) (*Observability, error) {
	obs := &Observability{
		KubeConfig:      kubeConfig,
		LightweightMode: false, // Default to full stack
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

func (o *Observability) SetLightweightMode(lightweight bool) {
	o.LightweightMode = lightweight
}

func (o *Observability) getChartValues() map[string]interface{} {
	if o.LightweightMode {
		return o.getLightweightChartValues()
	}
	return o.getFullChartValues()
}

func (o *Observability) getLightweightChartValues() map[string]interface{} {
	return map[string]interface{}{
		// Override the full name to keep resource names short
		"fullnameOverride": "obs",
		// Victoria Metrics configuration - minimal setup
		"vmsingle": map[string]interface{}{
			"enabled": true,
			"spec": map[string]interface{}{
				"retentionPeriod": "7d", // Shorter retention for lightweight
				"storage": map[string]interface{}{
					"storageClassName": "local-path",
					"accessModes":      []string{"ReadWriteOnce"},
					"size":             "5Gi", // Smaller storage
				},
			},
		},
		// Basic monitoring
		"nodeExporter": map[string]interface{}{
			"enabled": true,
		},
		"kubeStateMetrics": map[string]interface{}{
			"enabled": true,
		},
		// Grafana with basic configuration
		"grafana": map[string]interface{}{
			"enabled": true,
			"spec": map[string]interface{}{
				"storage": map[string]interface{}{
					"storageClassName": "local-path",
					"accessModes":      []string{"ReadWriteOnce"},
					"size":             "2Gi", // Smaller storage
				},
			},
			"sidecar": map[string]interface{}{
				"datasources": map[string]interface{}{
					"enabled": true,
				},
				"dashboards": map[string]interface{}{
					"enabled":         true,
					"label":           "grafana_dashboard",
					"searchNamespace": "ALL",
				},
			},
		},
		// Disable heavy components for lightweight mode
		"vlogs": map[string]interface{}{
			"enabled": false,
		},
		"fluent-bit": map[string]interface{}{
			"enabled": false,
		},
		"alertmanager": map[string]interface{}{
			"enabled": false,
		},
		"vmalert": map[string]interface{}{
			"enabled": false,
		},
		"jaeger": map[string]interface{}{
			"enabled": false,
		},
		"opentelemetry-collector": map[string]interface{}{
			"enabled": false,
		},
		// Essential service monitors
		"defaultRules": map[string]interface{}{
			"enabled": true,
			"create":  true,
			"rules": map[string]interface{}{
				"etcd":                        true,
				"general":                     true,
				"k8s":                         true,
				"kubeApiserver":               true,
				"kubeApiserverBurnrate":       true,
				"kubePrometheusGeneral":       true,
				"kubePrometheusNodeRecording": true,
				"kubernetesApps":              true,
				"kubernetesResources":         true,
				"kubernetesStorage":           true,
				"kubernetesSystem":            true,
				"node":                        true,
				"nodeExporter":                true,
				"prometheus":                  true,
				"prometheusOperator":          true,
			},
		},
		"prometheus": map[string]interface{}{
			"enabled": false,
		},
	}
}

func (o *Observability) getFullChartValues() map[string]interface{} {
	return map[string]interface{}{
		// Override the full name to keep resource names short
		"fullnameOverride": "obs",
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
		// Grafana configuration with comprehensive dashboards
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
					"enabled":                  true,
					"defaultDatasourceEnabled": true,
				},
				"dashboards": map[string]interface{}{
					"enabled":         true,
					"label":           "grafana_dashboard",
					"searchNamespace": "ALL",
					"provider": map[string]interface{}{
						"foldersFromFilesStructure": true,
					},
				},
			},
			"dashboardProviders": map[string]interface{}{
				"dashboardproviders.yaml": map[string]interface{}{
					"apiVersion": 1,
					"providers": []map[string]interface{}{
						{
							"name":            "default",
							"orgId":           1,
							"folder":          "",
							"type":            "file",
							"disableDeletion": false,
							"editable":        true,
							"options": map[string]interface{}{
								"path": "/var/lib/grafana/dashboards/default",
							},
						},
						{
							"name":            "cluster",
							"orgId":           1,
							"folder":          "Cluster",
							"type":            "file",
							"disableDeletion": false,
							"editable":        true,
							"options": map[string]interface{}{
								"path": "/var/lib/grafana/dashboards/cluster",
							},
						},
						{
							"name":            "applications",
							"orgId":           1,
							"folder":          "Applications",
							"type":            "file",
							"disableDeletion": false,
							"editable":        true,
							"options": map[string]interface{}{
								"path": "/var/lib/grafana/dashboards/applications",
							},
						},
					},
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
		// Comprehensive service monitors and rules
		"defaultRules": map[string]interface{}{
			"enabled": true,
			"create":  true,
			"rules": map[string]interface{}{
				"etcd":                        true,
				"general":                     true,
				"k8s":                         true,
				"kubeApiserver":               true,
				"kubeApiserverBurnrate":       true,
				"kubePrometheusGeneral":       true,
				"kubePrometheusNodeRecording": true,
				"kubernetesApps":              true,
				"kubernetesResources":         true,
				"kubernetesStorage":           true,
				"kubernetesSystem":            true,
				"node":                        true,
				"nodeExporter":                true,
				"prometheus":                  true,
				"prometheusOperator":          true,
			},
		},
		// Custom ServiceMonitors for application metrics
		"serviceMonitor": map[string]interface{}{
			"enabled": true,
		},
		// Prometheus compatibility disabled - using Victoria Metrics
		"prometheus": map[string]interface{}{
			"enabled": false,
		},
	}
}
