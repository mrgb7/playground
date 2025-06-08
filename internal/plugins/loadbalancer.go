package plugins

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mrgb7/playground/internal/k8s"
	"github.com/mrgb7/playground/pkg/logger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	repoURL      = "https://metallb.github.io/metallb"
	chartName    = "metallb"
	chartVersion = "0.14.9"
	releaseName  = "metallb"
	namespace    = "metallb-system"
	repoName     = "metallb"
)

type LoadBalancer struct {
	KubeConfig      string
	k8sClient       *k8s.K8sClient
	MasterClusterIP string
	ClusterName     string
	*BasePlugin
}

func NewLoadBalancer(kubeConfig string, masterClusterIP string, clusterName string) (*LoadBalancer, error) {
	c, err := k8s.NewK8sClient(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}
	lb := &LoadBalancer{
		KubeConfig:      kubeConfig,
		k8sClient:       c,
		MasterClusterIP: masterClusterIP,
		ClusterName:     clusterName,
	}
	lb.BasePlugin = NewBasePlugin(kubeConfig, lb)
	return lb, nil
}

func (l *LoadBalancer) GetOptions() PluginOptions {
	return PluginOptions{
		Version:          &chartVersion,
		Namespace:        &namespace,
		ChartName:        &chartName,
		RepoName:         &repoName,
		Repository:       &repoURL,
		releaseName:      &releaseName,
		CRDsGroupVersion: "metallb.io",
	}
}

func (l *LoadBalancer) GetName() string {
	return "load-balancer"
}

func (l *LoadBalancer) Install(kubeConfig, clusterName string, ensure ...bool) error {
	err := l.UnifiedInstall(kubeConfig, clusterName, ensure...)
	if err != nil {
		return fmt.Errorf("failed to install loadbalancer: %w", err)
	}

	err = l.ensure()
	if err != nil {
		return fmt.Errorf("failed to ensure loadbalancer: %w", err)
	}

	err = l.deleteValidationWebhookConfig()
	if err != nil {
		return fmt.Errorf("failed to delete validation webhook config: %w", err)
	}
	err = l.addl2IpPool()
	if err != nil {
		return fmt.Errorf("failed to add l2 ip pool: %w", err)
	}
	err = l.addl2Adv()
	if err != nil {
		return fmt.Errorf("failed to add l2 advertisement: %w", err)
	}
	return nil
}

func (l *LoadBalancer) ensure() error {
	timeout := 5 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for ensure %v", timeout)
		case <-ticker.C:
			_, err := l.k8sClient.GetNameSpace(namespace, ctx)
			if err != nil {
				continue
			}
			_, err = l.k8sClient.Clientset.
				AdmissionregistrationV1().
				ValidatingWebhookConfigurations().
				Get(ctx, "metallb-webhook-configuration", metav1.GetOptions{})
			if err != nil {
				continue
			}
			logger.Successln("LoadBalancer is ensured")
			return nil
		}
	}
}

func (l *LoadBalancer) Uninstall(kubeConfig, clusterName string, ensure ...bool) error {
	logger.Infoln("Uninstalling loadbalancer")
	return l.UnifiedUninstall(kubeConfig, clusterName, ensure...)
}

func (l *LoadBalancer) Status() string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ns, err := l.k8sClient.GetNameSpace(namespace, ctx)
	if ns == "" || err != nil {
		logger.Debugf("failed to get metallb namespace: %v", err)
		return StatusNotInstalled
	}

	return StatusRunning
}

func (l *LoadBalancer) addl2IpPool() error {
	ipRange := l.getIPRange()
	ipPooRes := schema.GroupVersionResource{
		Group:    "metallb.io",
		Version:  "v1beta1",
		Resource: "ipaddresspools",
	}

	ipPool := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "metallb.io/v1beta1",
			"kind":       "IPAddressPool",
			"metadata": map[string]interface{}{
				"name":      "k3s-pool-ip",
				"namespace": "metallb-system",
			},
			"spec": map[string]interface{}{
				"addresses": []interface{}{ipRange},
			},
		},
	}
	ipPool.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "metallb.io",
		Version: "v1beta1",
		Kind:    "IPAddressPool",
	})

	_, err := l.k8sClient.Dynamic.Resource(ipPooRes).
		Namespace(namespace).
		Create(context.TODO(), ipPool, metav1.CreateOptions{})

	switch {
	case err != nil && strings.Contains(err.Error(), "already exists"):
		// Get the existing IP address pool to preserve metadata
		existing, getErr := l.k8sClient.Dynamic.Resource(ipPooRes).
			Namespace(namespace).
			Get(context.TODO(), "k3s-pool-ip", metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("failed to get existing IP address pool: %w", getErr)
		}

		// Preserve the existing metadata and update only the spec
		ipPool.SetResourceVersion(existing.GetResourceVersion())
		ipPool.SetUID(existing.GetUID())
		ipPool.SetCreationTimestamp(existing.GetCreationTimestamp())
		ipPool.SetGeneration(existing.GetGeneration())

		// Copy any existing labels and annotations
		if labels := existing.GetLabels(); labels != nil {
			ipPool.SetLabels(labels)
		}
		if annotations := existing.GetAnnotations(); annotations != nil {
			ipPool.SetAnnotations(annotations)
		}

		_, err = l.k8sClient.Dynamic.Resource(ipPooRes).
			Namespace(namespace).
			Update(context.TODO(), ipPool, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update existing IP address pool: %w", err)
		}
		logger.Infoln("Updated existing IP address pool")
	case err != nil:
		logger.Errorln("failed to create ip address pool: %v", err)
		return fmt.Errorf("failed to create ip address pool: %w", err)
	default:
		logger.Successln("Created IP address pool successfully")
	}
	return nil
}

func (l *LoadBalancer) addl2Adv() error {
	l2AdvRes := schema.GroupVersionResource{
		Group:    "metallb.io",
		Version:  "v1beta1",
		Resource: "l2advertisements",
	}

	l2Adv := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "metallb.io/v1beta1",
			"kind":       "L2Advertisement",
			"metadata": map[string]interface{}{
				"name":      "k3s-lb-pool",
				"namespace": "metallb-system",
			},
			"spec": map[string]interface{}{
				"ipAddressPools": []interface{}{"k3s-pool-ip"},
			},
		},
	}
	l2Adv.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "metallb.io",
		Version: "v1beta1",
		Kind:    "L2Advertisement",
	})

	_, err := l.k8sClient.Dynamic.Resource(l2AdvRes).
		Namespace(namespace).
		Create(context.TODO(), l2Adv, metav1.CreateOptions{})

	switch {
	case err != nil && strings.Contains(err.Error(), "already exists"):
		// Get the existing L2Advertisement to preserve metadata
		existing, getErr := l.k8sClient.Dynamic.Resource(l2AdvRes).
			Namespace(namespace).
			Get(context.TODO(), "k3s-lb-pool", metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("failed to get existing L2Advertisement: %w", getErr)
		}

		// Preserve the existing metadata and update only the spec
		l2Adv.SetResourceVersion(existing.GetResourceVersion())
		l2Adv.SetUID(existing.GetUID())
		l2Adv.SetCreationTimestamp(existing.GetCreationTimestamp())
		l2Adv.SetGeneration(existing.GetGeneration())

		// Copy any existing labels and annotations
		if labels := existing.GetLabels(); labels != nil {
			l2Adv.SetLabels(labels)
		}
		if annotations := existing.GetAnnotations(); annotations != nil {
			l2Adv.SetAnnotations(annotations)
		}

		_, err = l.k8sClient.Dynamic.Resource(l2AdvRes).
			Namespace(namespace).
			Update(context.TODO(), l2Adv, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update existing L2Advertisement: %w", err)
		}
		logger.Infoln("Updated existing L2Advertisement")
	case err != nil:
		logger.Errorln("failed to create l2 advertisement: %v", err)
		return fmt.Errorf("failed to create l2 advertisement: %w", err)
	default:
		logger.Successln("Created L2Advertisement successfully")
	}
	return nil
}

func (l *LoadBalancer) deleteValidationWebhookConfig() error {
	err := l.k8sClient.Clientset.AdmissionregistrationV1().
		ValidatingWebhookConfigurations().
		Delete(context.Background(), "metallb-webhook-configuration", metav1.DeleteOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			logger.Debugln("Validation webhook configuration already deleted")
			return nil // Already deleted
		}
		return fmt.Errorf("failed to delete validation webhook configuration: %w", err)
	}

	return nil
}

func (l *LoadBalancer) getIPRange() string {
	ipParts := strings.Split(l.MasterClusterIP, ".")
	dhcp := ipParts[:3]
	
	// Use cluster name to determine IP range offset to avoid conflicts
	clusterOffset := l.getClusterOffset()
	
	// Start from 100 and allocate 5 IPs per cluster to support more clusters
	// This allows for 31 clusters (100-254 range with 5 IPs each)
	baseStart := 100
	rangeSize := 5
	
	start := baseStart + (clusterOffset * rangeSize)
	end := start + rangeSize - 1
	
	// Ensure we don't exceed 254 (keeping 255 reserved)
	if end > 254 {
		// Fallback to a smaller range if we're near the limit
		start = 250
		end = 254
		logger.Warnln("Cluster %s: IP range limited due to address space constraints", l.ClusterName)
	}
	
	startIP := fmt.Sprintf("%s.%d", strings.Join(dhcp, "."), start)
	endIP := fmt.Sprintf("%s.%d", strings.Join(dhcp, "."), end)
	return fmt.Sprintf("%s-%s", startIP, endIP)
}

// getClusterOffset generates a consistent offset based on cluster name
// Returns a value between 0-30 to support up to 31 clusters with 5 IPs each
func (l *LoadBalancer) getClusterOffset() int {
	// Simple hash function to get a deterministic offset from cluster name
	hash := 0
	for _, c := range l.ClusterName {
		hash += int(c)
	}
	// Return offset between 0-30 to allow for 31 clusters with 5 IPs each (100-254)
	return hash % 31
}

func (l *LoadBalancer) GetDependencies() []string {
	return []string{}
}
