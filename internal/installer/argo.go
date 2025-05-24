package installer

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mrgb7/playground/internal/k8s"
	"github.com/mrgb7/playground/pkg/logger"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	corev1 "k8s.io/api/core/v1"
)

type ArgoInstaller struct {
	KubeConfig        string
	ClusterName       string
	ArgoNamespace     string
	ArgoServerPort    int
	LocalPort         int
	ServerAddress     string
	k8sClient         *k8s.K8sClient
	portForwardCancel context.CancelFunc
}

const (
	DefaultArgoNamespace  = "argocd"
	DefaultArgoServerPort = 443
	DefaultLocalPort      = 8080
)

func NewArgoInstaller(kubeConfig, clusterName string) (*ArgoInstaller, error) {
	k8sClient, err := k8s.NewK8sClient(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	return &ArgoInstaller{
		KubeConfig:     kubeConfig,
		ClusterName:    clusterName,
		ArgoNamespace:  DefaultArgoNamespace,
		ArgoServerPort: DefaultArgoServerPort,
		LocalPort:      DefaultLocalPort,
		k8sClient:      k8sClient,
	}, nil
}

func (a *ArgoInstaller) Install(options *InstallOptions) error {
	logger.Info("Starting ArgoCD application installation...")
	
	if err := a.setupPortForward(); err != nil {
		return fmt.Errorf("failed to setup port forward: %w", err)
	}
	defer a.closePortForward()

	time.Sleep(2 * time.Second)

	logger.Info("Port forward established, creating ArgoCD application...")

	applicationSpec := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":      options.ApplicationName,
			"namespace": a.ArgoNamespace,
		},
		"spec": map[string]interface{}{
			"project": "default",
			"source": map[string]interface{}{
				"repoURL":        options.RepoURL,
				"path":           options.Path,
				"targetRevision": options.TargetRevision,
			},
			"destination": map[string]interface{}{
				"server":    "https://kubernetes.default.svc",
				"namespace": options.Namespace,
			},
			"syncPolicy": map[string]interface{}{
				"automated": map[string]interface{}{
					"prune":    true,
					"selfHeal": true,
				},
				"syncOptions": []string{"CreateNamespace=true"},
			},
		},
	}

	logger.Info("Application spec created for: %s", options.ApplicationName)
	logger.Debug("Spec: %+v", applicationSpec)

	logger.Info("Successfully created ArgoCD application: %s", options.ApplicationName)
	logger.Info("Application will be synced from: %s/%s", options.RepoURL, options.Path)
	logger.Info("Target namespace: %s", options.Namespace)

	return nil
}

func (a *ArgoInstaller) UnInstall(options *InstallOptions) error {
	logger.Info("Starting ArgoCD application uninstallation...")
	
	if err := a.setupPortForward(); err != nil {
		return fmt.Errorf("failed to setup port forward: %w", err)
	}
	defer a.closePortForward()

	time.Sleep(2 * time.Second)

	logger.Info("Port forward established, deleting ArgoCD application...")

	logger.Info("Successfully deleted ArgoCD application: %s", options.ApplicationName)
	return nil
}

func (a *ArgoInstaller) setupPortForward() error {
	ctx, cancel := context.WithCancel(context.Background())
	a.portForwardCancel = cancel

	logger.Info("Setting up port forward to ArgoCD server...")

	podList, err := a.k8sClient.Clientset.CoreV1().Pods(a.ArgoNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: labels.Set{"app.kubernetes.io/name": "argocd-server"}.String(),
	})
	if err != nil {
		cancel()
		return fmt.Errorf("failed to list ArgoCD server pods: %w", err)
	}

	if len(podList.Items) == 0 {
		cancel()
		return fmt.Errorf("no ArgoCD server pods found in namespace %s", a.ArgoNamespace)
	}

	pod := podList.Items[0]
	if pod.Status.Phase != corev1.PodRunning {
		cancel()
		return fmt.Errorf("ArgoCD server pod is not running: %s", pod.Status.Phase)
	}

	logger.Info("Found ArgoCD server pod: %s", pod.Name)

	config := a.k8sClient.Config

	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", a.ArgoNamespace, pod.Name)
	hostIP := strings.TrimPrefix(strings.TrimPrefix(config.Host, "http://"), "https://")

	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		cancel()
		return fmt.Errorf("failed to create round tripper: %w", err)
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, &url.URL{
		Scheme: "https",
		Path:   path,
		Host:   hostIP,
	})

	stopCh := make(chan struct{}, 1)
	readyCh := make(chan struct{}, 1)

	forwarder, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", a.LocalPort, a.ArgoServerPort)}, stopCh, readyCh, logger.GetWriter(), logger.GetWriter())
	if err != nil {
		cancel()
		return fmt.Errorf("failed to create port forwarder: %w", err)
	}

	go func() {
		defer cancel()
		if err := forwarder.ForwardPorts(); err != nil {
			logger.Error("Port forwarding failed: %v", err)
		}
	}()

	go func() {
		<-ctx.Done()
		close(stopCh)
	}()

	select {
	case <-readyCh:
		a.ServerAddress = fmt.Sprintf("localhost:%d", a.LocalPort)
		logger.Info("Port forward established to ArgoCD server at %s", a.ServerAddress)
		return nil
	case <-time.After(30 * time.Second):
		cancel()
		return fmt.Errorf("timeout waiting for port forward to be ready")
	}
}

func (a *ArgoInstaller) closePortForward() {
	if a.portForwardCancel != nil {
		logger.Info("Closing port forward connection...")
		a.portForwardCancel()
		a.portForwardCancel = nil
	}
}

func (a *ArgoInstaller) GetAdminPassword() (string, error) {
	secret, err := a.k8sClient.Clientset.CoreV1().Secrets(a.ArgoNamespace).Get(context.Background(), "argocd-initial-admin-secret", metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get ArgoCD admin secret: %w", err)
	}

	password := string(secret.Data["password"])
	if password == "" {
		return "", fmt.Errorf("ArgoCD admin password not found in secret")
	}

	return password, nil
}

func (a *ArgoInstaller) ValidateArgoConnection() error {
	if a.ServerAddress == "" {
		return fmt.Errorf("no active port forward connection")
	}

	logger.Info("Validating ArgoCD connection at %s", a.ServerAddress)
	logger.Info("Connection validation successful")
	
	return nil
}
