package plugins

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mrgb7/playground/internal/k8s"
	"github.com/mrgb7/playground/pkg/logger"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	IngressNamespace = "ingress-system"
	IngressName      = "ingress"
	IngressVersion   = "1.0.0"
	ArgoCDPort       = 80
)

type Ingress struct {
	KubeConfig  string
	k8sClient   *k8s.K8sClient
	ClusterName string
	*BasePlugin
}

func NewIngress(kubeConfig, clusterName string) (*Ingress, error) {
	c, err := k8s.NewK8sClient(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}
	ingress := &Ingress{
		KubeConfig:  kubeConfig,
		k8sClient:   c,
		ClusterName: clusterName,
	}
	ingress.BasePlugin = NewBasePlugin(kubeConfig, ingress)
	return ingress, nil
}

func (i *Ingress) GetName() string {
	return IngressName
}

func (i *Ingress) Install(kubeConfig, clusterName string, ensure ...bool) error {
	logger.Infoln("Installing ingress plugin for cluster: %s", clusterName)

	if err := i.checkDependencies(); err != nil {
		return fmt.Errorf("dependency check failed: %w", err)
	}

	if err := i.ensureNginxLoadBalancer(); err != nil {
		return fmt.Errorf("failed to ensure nginx LoadBalancer: %w", err)
	}

	i.setupClusterDomain()

	if err := i.configureArgoCDIngress(); err != nil {
		return fmt.Errorf("failed to configure ArgoCD ingress: %w", err)
	}

	if err := i.printHostInstructions(); err != nil {
		return fmt.Errorf("failed to print host instructions: %w", err)
	}

	logger.Successln("Ingress plugin installed successfully")
	return nil
}

func (i *Ingress) Uninstall(kubeConfig, clusterName string, ensure ...bool) error {
	logger.Infoln("Uninstalling ingress plugin")

	err := i.removeArgoCDIngress()
	if err != nil {
		logger.Warnln("Failed to remove ArgoCD ingress: %v", err)
	}

	logger.Successln("Ingress plugin uninstalled successfully")
	return nil
}

func (i *Ingress) Status() string {
	nginx := NewNginx(i.KubeConfig)
	lb, _ := NewLoadBalancer(i.KubeConfig, "")

	nginxStatus := nginx.Status()
	lbStatus := lb.Status()

	if !strings.Contains(nginxStatus, StatusRunning) || !strings.Contains(lbStatus, StatusRunning) {
		return "Ingress dependencies not satisfied"
	}

	return "Ingress is configured"
}

func (i *Ingress) checkDependencies() error {
	logger.Infoln("Checking ingress dependencies...")

	nginx := NewNginx(i.KubeConfig)
	nginxStatus := nginx.Status()
	if !strings.Contains(nginxStatus, StatusRunning) {
		return fmt.Errorf("nginx plugin is required but not installed/running. Status: %s", nginxStatus)
	}

	lb, err := NewLoadBalancer(i.KubeConfig, "")
	if err != nil {
		return fmt.Errorf("failed to create loadbalancer client: %w", err)
	}
	lbStatus := lb.Status()
	if !strings.Contains(lbStatus, StatusRunning) {
		return fmt.Errorf("loadbalancer plugin is required but not installed/running. Status: %s", lbStatus)
	}

	logger.Successln("All dependencies satisfied")
	return nil
}

func (i *Ingress) ensureNginxLoadBalancer() error {
	logger.Infoln("Ensuring nginx service is LoadBalancer type...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	svc, err := i.k8sClient.Clientset.CoreV1().Services(NginxNamespace).Get(
		ctx, "nginx-ingress-ingress-nginx-controller", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get nginx service: %w", err)
	}

	if svc.Spec.Type == v1.ServiceTypeLoadBalancer {
		logger.Debugln("Nginx service is already LoadBalancer type")
		return nil
	}

	svc.Spec.Type = v1.ServiceTypeLoadBalancer
	_, err = i.k8sClient.Clientset.CoreV1().Services(NginxNamespace).Update(ctx, svc, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update nginx service to LoadBalancer: %w", err)
	}

	logger.Successln("Updated nginx service to LoadBalancer type")
	return nil
}

func (i *Ingress) setupClusterDomain() {
	logger.Infoln("Setting up cluster domain: %s.local", i.ClusterName)
}

func (i *Ingress) configureArgoCDIngress() error {
	logger.Infoln("Checking for ArgoCD installation...")

	argocd := NewArgocd(i.KubeConfig)
	argoCDStatus := argocd.Status()
	if !strings.Contains(argoCDStatus, StatusRunning) {
		logger.Infoln("ArgoCD not installed, skipping ingress configuration")
		return nil
	}

	logger.Infoln("ArgoCD found, configuring ingress...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	existingIngress, err := i.k8sClient.Clientset.NetworkingV1().Ingresses("argocd").Get(
		ctx, "argocd-server", metav1.GetOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return fmt.Errorf("failed to check existing ArgoCD ingress: %w", err)
	}

	if err == nil {
		logger.Infoln("Updating existing ArgoCD ingress with cluster domain...")

		if len(existingIngress.Spec.Rules) > 0 {
			existingIngress.Spec.Rules[0].Host = fmt.Sprintf("argocd.%s.local", i.ClusterName)
		}

		if len(existingIngress.Spec.TLS) > 0 && len(existingIngress.Spec.TLS[0].Hosts) > 0 {
			existingIngress.Spec.TLS[0].Hosts[0] = fmt.Sprintf("argocd.%s.local", i.ClusterName)
		}

		_, err = i.k8sClient.Clientset.NetworkingV1().Ingresses("argocd").Update(ctx, existingIngress, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update existing ArgoCD ingress: %w", err)
		}
		logger.Successln("Updated existing ArgoCD ingress with host: argocd.%s.local", i.ClusterName)
	} else {
		logger.Infoln("Creating new ArgoCD ingress...")

		ingress := &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "argocd-server",
				Namespace: "argocd",
				Annotations: map[string]string{
					"nginx.ingress.kubernetes.io/ssl-redirect":       "false",
					"nginx.ingress.kubernetes.io/force-ssl-redirect": "false",
					"nginx.ingress.kubernetes.io/backend-protocol":   "HTTP",
				},
			},
			Spec: networkingv1.IngressSpec{
				IngressClassName: func() *string { s := "nginx"; return &s }(),
				Rules: []networkingv1.IngressRule{
					{
						Host: fmt.Sprintf("argocd.%s.local", i.ClusterName),
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									{
										Path:     "/",
										PathType: func() *networkingv1.PathType { pt := networkingv1.PathTypePrefix; return &pt }(),
										Backend: networkingv1.IngressBackend{
											Service: &networkingv1.IngressServiceBackend{
												Name: "argocd-server",
												Port: networkingv1.ServiceBackendPort{
													Number: ArgoCDPort,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		_, err = i.k8sClient.Clientset.NetworkingV1().Ingresses("argocd").Create(ctx, ingress, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create ArgoCD ingress: %w", err)
		}
		logger.Successln("Created ArgoCD ingress with host: argocd.%s.local", i.ClusterName)
	}

	return nil
}

func (i *Ingress) removeArgoCDIngress() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := i.k8sClient.Clientset.NetworkingV1().Ingresses("argocd").Delete(
		ctx, "argocd-server", metav1.DeleteOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return fmt.Errorf("failed to delete ArgoCD ingress: %w", err)
	}

	return nil
}

func (i *Ingress) printHostInstructions() error {
	logger.Infoln("Getting nginx LoadBalancer IP...")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var nginxIP string
	for retries := 0; retries < 12; retries++ {
		svc, err := i.k8sClient.Clientset.CoreV1().Services(NginxNamespace).Get(
			ctx, "nginx-ingress-ingress-nginx-controller", metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get nginx service: %w", err)
		}

		if len(svc.Status.LoadBalancer.Ingress) > 0 {
			if svc.Status.LoadBalancer.Ingress[0].IP != "" {
				nginxIP = svc.Status.LoadBalancer.Ingress[0].IP
				break
			}
		}

		logger.Infoln("Waiting for LoadBalancer IP assignment... (%d/12)", retries+1)
		time.Sleep(5 * time.Second)
	}

	if nginxIP == "" {
		logger.Warnln("LoadBalancer IP not available yet. You can run this command later to get it:")
		logger.Infoln("kubectl get svc -n %s nginx-ingress-ingress-nginx-controller "+
			"-o jsonpath='{.status.loadBalancer.ingress[0].ip}'", NginxNamespace)
		return nil
	}

	logger.Successln("LoadBalancer IP found: %s", nginxIP)
	logger.Infoln("")
	logger.Infoln("🎯 Add these entries to your /etc/hosts file:")
	logger.Infoln("echo '%s %s.local' | sudo tee -a /etc/hosts", nginxIP, i.ClusterName)

	argocd := NewArgocd(i.KubeConfig)
	argoCDStatus := argocd.Status()
	if strings.Contains(argoCDStatus, StatusRunning) {
		logger.Infoln("echo '%s argocd.%s.local' | sudo tee -a /etc/hosts", nginxIP, i.ClusterName)
		logger.Infoln("")
		logger.Infoln("🚀 ArgoCD will be available at: http://argocd.%s.local", i.ClusterName)
	}

	logger.Infoln("")
	logger.Infoln("🌐 Cluster domain: %s.local", i.ClusterName)

	return nil
}

func (i *Ingress) GetNamespace() string {
	return IngressNamespace
}

func (i *Ingress) GetVersion() string {
	return IngressVersion
}

func (i *Ingress) GetChartName() string {
	return ""
}

func (i *Ingress) GetRepository() string {
	return ""
}

func (i *Ingress) GetRepoName() string {
	return ""
}

func (i *Ingress) GetChartValues() map[string]interface{} {
	return make(map[string]interface{})
}
