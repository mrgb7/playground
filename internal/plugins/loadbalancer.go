package plugins

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mrgb7/playground/internal/installer"
	"github.com/mrgb7/playground/internal/k8s"
	"github.com/mrgb7/playground/pkg/logger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	repoUrl      = "https://metallb.github.io/metallb"
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
	*BasePlugin
}

func NewLoadBalancer(kubeConfig string, masterClusterIP string) (*LoadBalancer, error) {
	c, err := k8s.NewK8sClient(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}
	lb := &LoadBalancer{
		KubeConfig:      kubeConfig,
		k8sClient:       c,
		MasterClusterIP: masterClusterIP,
	}
	lb.BasePlugin = NewBasePlugin(kubeConfig, lb)
	return lb, nil
}

func (l *LoadBalancer) GetName() string {
	return "loadBalancer"
}

func (l *LoadBalancer) GetInstaller() (installer.Installer, error) {
	return &installer.HelmInstaller{
		ReleaseName:  releaseName,
		ChartName:    chartName,
		RepoUrl:      repoUrl,
		RepoName:     repoName,
		Namespace:    namespace,
		ChartVersion: chartVersion,
		KubeConfig:   l.KubeConfig,
	}, nil
}

func (l *LoadBalancer) Install(ensure ...bool) error {
	i, err := l.GetInstaller()
	if err != nil {
		return fmt.Errorf("failed to get installer: %w", err)
	}
	err = i.Install(&installer.InstallOptions{})
	if err != nil {
		return fmt.Errorf("failed to install loadbalancer: %w", err)
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

func (l *LoadBalancer) Uninstall(ensure ...bool) error {
	fmt.Println("Uninstalling loadbalancer")
	i, err := l.GetInstaller()
	if err != nil {
		return fmt.Errorf("failed to get installer: %w", err)
	}
	err = i.UnInstall(&installer.InstallOptions{})
	if err != nil {
		return fmt.Errorf("failed to uninstall loadbalancer: %w", err)
	}

	return nil
}

func (l *LoadBalancer) Status() string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ns, err := l.k8sClient.GetNameSpace(namespace, ctx)
	if ns == "" || err != nil {
		logger.Error("failed to get metallb namespace: %v", err)
		return "Not installed"
	}

	return "LoadBalancer is running"
}

func (l *LoadBalancer) addl2IpPool() error {
	ipRange, err := l.getIPRange()
	if err != nil {
		logger.Error("failed to get ip range: %v", err)
		return fmt.Errorf("failed to get ip range: %w", err)
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
	ipPooRes := schema.GroupVersionResource{
		Group:    "metallb.io",
		Version:  "v1beta1",
		Resource: "ipaddresspools",
	}
	_, err = l.k8sClient.Dynamic.Resource(ipPooRes).
		Namespace(namespace).
		Create(context.TODO(), ipPool, metav1.CreateOptions{})
	if err != nil {
		logger.Error("failed to create ip address pool: %v", err)
		return fmt.Errorf("failed to create ip address pool: %w", err)
	}
	return nil
}

func (l *LoadBalancer) addl2Adv() error {
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

	l2AdvRes := schema.GroupVersionResource{
		Group:    "metallb.io",
		Version:  "v1beta1",
		Resource: "l2advertisements",
	}

	_, err := l.k8sClient.Dynamic.Resource(l2AdvRes).
		Namespace(namespace).
		Create(context.TODO(), l2Adv, metav1.CreateOptions{})
	if err != nil {
		logger.Error("failed to create l2 advertisement: %v", err)
		return fmt.Errorf("failed to create l2 advertisement: %w", err)
	}
	return nil
}

func (l *LoadBalancer) deleteValidationWebhookConfig() error {
	return l.k8sClient.Clientset.AdmissionregistrationV1().
		ValidatingWebhookConfigurations().
		Delete(context.TODO(), "metallb-webhook-configuration", metav1.DeleteOptions{})
}

func (l *LoadBalancer) getIPRange() (string, error) {
	ipParts := strings.Split(l.MasterClusterIP, ".")
	dhcp := ipParts[:3]
	start := fmt.Sprintf("%s.100", strings.Join(dhcp, "."))
	end := fmt.Sprintf("%s.200", strings.Join(dhcp, "."))
	return fmt.Sprintf("%s-%s", start, end), nil
}
