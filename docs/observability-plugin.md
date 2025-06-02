# Observability Plugin

The observability plugin provides comprehensive monitoring capabilities for Kubernetes clusters created with playground, including metrics collection, distributed tracing, log aggregation, and visualization dashboards.

## Features

### Metrics Collection
- **Victoria Metrics Single**: High-performance, cost-effective metrics storage
- **Node Exporter**: Node-level metrics collection
- **Kube State Metrics**: Kubernetes object metrics
- **Custom ServiceMonitors**: Application metrics collection

### Distributed Tracing
- **Jaeger**: Distributed tracing system for microservices
- **OpenTelemetry Collector**: Trace collection and processing
- **Trace ingestion and storage**: Complete tracing pipeline

### Log Aggregation
- **Victoria Logs**: Fast, efficient log storage and indexing
- **Fluent Bit**: Lightweight log processor and forwarder
- **Log parsing and enrichment**: Structured log processing

### Alerting
- **AlertManager**: Alert routing and notification management
- **Victoria Metrics Alert**: Alerting rules engine

### Visualization
- **Grafana**: Comprehensive dashboarding and visualization
- **Pre-configured dashboards**: Ready-to-use monitoring dashboards
- **Automatic datasource configuration**: Zero-configuration setup

## Installation Options

The observability plugin supports two installation modes:

### Full Stack (Default)
Includes all observability components for comprehensive monitoring:

```bash
playground cluster plugin add --name observability --cluster <cluster-name>
```

### Lightweight Mode
Minimal monitoring setup with reduced resource usage:

```bash
# Note: Lightweight mode requires manual configuration via plugin options
# This is configured at the code level for now
```

## Configuration

### Full Stack Configuration
- **Metrics retention**: 30 days
- **Storage**: 20Gi for metrics, 10Gi for logs, 5Gi for Grafana, 2Gi for alerts
- **Components**: All components enabled
- **Dashboards**: Comprehensive dashboard suite organized by categories

### Lightweight Configuration
- **Metrics retention**: 7 days
- **Storage**: 5Gi for metrics, 2Gi for Grafana
- **Components**: Basic metrics, node monitoring, and Grafana only
- **Dashboards**: Essential monitoring dashboards

## Usage

### Basic Installation

```bash
# Install observability plugin
playground cluster plugin add --name observability --cluster my-cluster

# Install ingress plugin to expose UIs (optional but recommended)
playground cluster plugin add --name ingress --cluster my-cluster

# Add host entries (provided by ingress plugin)
echo '<LoadBalancer-IP> grafana.my-cluster.local' | sudo tee -a /etc/hosts
echo '<LoadBalancer-IP> victoria-metrics.my-cluster.local' | sudo tee -a /etc/hosts
echo '<LoadBalancer-IP> victoria-logs.my-cluster.local' | sudo tee -a /etc/hosts
echo '<LoadBalancer-IP> jaeger.my-cluster.local' | sudo tee -a /etc/hosts
```

### Accessing Components

After installation and ingress configuration:

- **Grafana**: `http://grafana.<cluster-name>.local`
- **Victoria Metrics**: `http://victoria-metrics.<cluster-name>.local`
- **Victoria Logs**: `http://victoria-logs.<cluster-name>.local`
- **Jaeger**: `http://jaeger.<cluster-name>.local`

With TLS plugin installed:
- **Grafana**: `https://grafana.<cluster-name>.local`
- And so on...

### Kubernetes Access

Direct access via kubectl port-forward:

```bash
# Grafana
kubectl port-forward -n monitoring svc/grafana 3000:3000

# Victoria Metrics
kubectl port-forward -n monitoring svc/vmsingle-observability 8428:8428

# Jaeger Query
kubectl port-forward -n monitoring svc/jaeger-query 16686:16686
```

## Pre-configured Dashboards

The plugin includes organized dashboards for:

### Cluster Overview
- Cluster resource utilization
- Node health and performance
- Pod resource consumption
- Network and storage metrics

### Application Monitoring
- Application performance metrics
- Custom metrics from ServiceMonitors
- Service-level indicators

### System Monitoring
- Kubernetes system components
- etcd performance
- API server metrics
- Controller manager metrics

## Architecture

### Components Layout

```
┌─────────────────────────────────────────────────────────────┐
│                    Monitoring Namespace                     │
│                                                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │   Grafana   │  │  Victoria   │  │   Jaeger    │         │
│  │             │  │   Metrics   │  │   Query     │         │
│  │ Dashboards  │  │   Single    │  │             │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
│                                                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │ AlertManager│  │  Victoria   │  │ OpenTelemetry│         │
│  │             │  │    Logs     │  │  Collector  │         │
│  │             │  │             │  │             │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
│                                                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │ Node        │  │ Kube State  │  │ Fluent Bit  │         │
│  │ Exporter    │  │ Metrics     │  │             │         │
│  │             │  │             │  │             │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
└─────────────────────────────────────────────────────────────┘
```

### Data Flow

1. **Metrics Collection**: Node Exporter and Kube State Metrics collect cluster metrics
2. **Metrics Storage**: Victoria Metrics Single stores and indexes metrics
3. **Log Collection**: Fluent Bit collects logs from all pods
4. **Log Storage**: Victoria Logs processes and stores logs
5. **Trace Collection**: OpenTelemetry Collector receives traces from applications
6. **Trace Storage**: Jaeger stores and indexes distributed traces
7. **Visualization**: Grafana queries all data sources for unified dashboards
8. **Alerting**: AlertManager processes alerts from Victoria Metrics Alert rules

## Storage Requirements

### Full Stack
- **Total Storage**: ~37Gi
  - Victoria Metrics: 20Gi
  - Victoria Logs: 10Gi
  - Grafana: 5Gi
  - AlertManager: 2Gi

### Lightweight Stack
- **Total Storage**: ~7Gi
  - Victoria Metrics: 5Gi
  - Grafana: 2Gi

## Resource Requirements

### Full Stack
- **CPU**: ~1-2 cores total
- **Memory**: ~2-4Gi total
- **Storage**: ~37Gi total

### Lightweight Stack
- **CPU**: ~0.5-1 cores total
- **Memory**: ~1-2Gi total
- **Storage**: ~7Gi total

## Integration with Other Plugins

### Ingress Plugin
The ingress plugin automatically configures ingress routes for all observability components when both plugins are installed:

- Automatic subdomain creation (`grafana.<cluster>.local`)
- TLS certificate generation (when TLS plugin is installed)
- Host file instructions for local development

### TLS Plugin
When the TLS plugin is installed:
- Automatic HTTPS configuration for all observability UIs
- Certificate auto-renewal via cert-manager
- Secure access to all monitoring tools

## Troubleshooting

### Plugin Installation Issues

```bash
# Check plugin status
playground cluster plugin list --cluster <cluster-name>

# Check namespace creation
kubectl get namespace monitoring

# Check pod status
kubectl get pods -n monitoring
```

### Component Issues

```bash
# Check Victoria Metrics
kubectl logs -n monitoring statefulset/vmsingle-observability

# Check Grafana
kubectl logs -n monitoring deployment/grafana

# Check ingress (if installed)
kubectl get ingress -n monitoring
```

### Resource Issues

```bash
# Check resource usage
kubectl top pods -n monitoring
kubectl top nodes

# Check storage
kubectl get pvc -n monitoring
```

## Customization

### Adding Custom Dashboards

1. Create a ConfigMap with your dashboard JSON:
```bash
kubectl create configmap my-dashboard --from-file=dashboard.json -n monitoring
kubectl label configmap my-dashboard grafana_dashboard="1" -n monitoring
```

2. Grafana will automatically pick up the new dashboard

### Adding Custom ServiceMonitors

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: my-app-monitor
  namespace: monitoring
spec:
  selector:
    matchLabels:
      app: my-app
  endpoints:
  - port: metrics
    path: /metrics
```

## Uninstallation

```bash
# Remove observability plugin
playground cluster plugin remove --name observability --cluster <cluster-name>

# Clean up any remaining resources (if needed)
kubectl delete namespace monitoring
```

## Support

For issues and feature requests, please refer to the [main project repository](https://github.com/mrgb7/playground/issues). 