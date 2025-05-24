package installer

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mrgb7/playground/internal/k8s"
	"github.com/mrgb7/playground/pkg/logger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type ArgoInstaller struct {
	KubeConfig        string
	ClusterName       string
	ArgoNamespace     string
	ArgoServerPort    int
	LocalPort         int
	ServerAddress     string
	k8sClient         *k8s.K8sClient
	httpClient        *http.Client
	authToken         string
	stopChannel       chan struct{}
	readyChannel      chan struct{}
}

type ArgoApplication struct {
	APIVersion string             `json:"apiVersion"`
	Kind       string             `json:"kind"`
	Metadata   ArgoMetadata       `json:"metadata"`
	Spec       ArgoApplicationSpec `json:"spec"`
}

type ArgoMetadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type ArgoApplicationSpec struct {
	Project     string              `json:"project"`
	Source      ArgoSource          `json:"source"`
	Destination ArgoDestination     `json:"destination"`
	SyncPolicy  *ArgoSyncPolicy     `json:"syncPolicy,omitempty"`
}

type ArgoSource struct {
	RepoURL        string `json:"repoURL"`
	Path           string `json:"path"`
	TargetRevision string `json:"targetRevision"`
}

type ArgoDestination struct {
	Server    string `json:"server"`
	Namespace string `json:"namespace"`
}

type ArgoSyncPolicy struct {
	Automated   *ArgoSyncPolicyAutomated `json:"automated,omitempty"`
	SyncOptions []string                 `json:"syncOptions,omitempty"`
}

type ArgoSyncPolicyAutomated struct {
	Prune    bool `json:"prune,omitempty"`
	SelfHeal bool `json:"selfHeal,omitempty"`
}

type ArgoSessionRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type ArgoSessionResponse struct {
	Token string `json:"token"`
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

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	return &ArgoInstaller{
		KubeConfig:     kubeConfig,
		ClusterName:    clusterName,
		ArgoNamespace:  DefaultArgoNamespace,
		ArgoServerPort: DefaultArgoServerPort,
		LocalPort:      DefaultLocalPort,
		k8sClient:      k8sClient,
		httpClient:     httpClient,
	}, nil
}

func (a *ArgoInstaller) Install(options *InstallOptions) error {
	if options == nil {
		return fmt.Errorf("install options cannot be nil")
	}
	
	logger.Info("Starting ArgoCD application installation...\n")
	
	if err := a.connectToArgoCD(); err != nil {
		return fmt.Errorf("failed to connect to ArgoCD: %w", err)
	}
	defer a.cleanup()

	if err := a.createApplication(options); err != nil {
		return fmt.Errorf("failed to create ArgoCD application: %w", err)
	}

	logger.Info("Successfully created ArgoCD application: %s", options.ApplicationName)
	return nil
}

func (a *ArgoInstaller) UnInstall(options *InstallOptions) error {
	if options == nil {
		return fmt.Errorf("install options cannot be nil")
	}
	
	logger.Info("Starting ArgoCD application uninstallation...\n")
	
	if err := a.connectToArgoCD(); err != nil {
		return fmt.Errorf("failed to connect to ArgoCD: %w", err)
	}
	defer a.cleanup()

	if err := a.deleteApplication(options); err != nil {
		return fmt.Errorf("failed to delete ArgoCD application: %w", err)
	}

	logger.Info("Successfully deleted ArgoCD application: %s", options.ApplicationName)
	return nil
}

func (a *ArgoInstaller) connectToArgoCD() error {
	password, err := a.GetAdminPassword()
	if err != nil {
		return fmt.Errorf("failed to get admin password: %w", err)
	}

	if err := a.setupPortForward(); err != nil {
		return fmt.Errorf("failed to setup port forward: %w", err)
	}

	time.Sleep(2 * time.Second)

	return a.authenticate(password)
}

func (a *ArgoInstaller) authenticate(password string) error {
	sessionReq := ArgoSessionRequest{
		Username: "admin",
		Password: password,
	}

	reqBody, err := json.Marshal(sessionReq)
	if err != nil {
		return fmt.Errorf("failed to marshal session request: %w", err)
	}

	url := fmt.Sprintf("https://%s/api/v1/session", a.ServerAddress)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create session request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authentication failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	var sessionResp ArgoSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&sessionResp); err != nil {
		return fmt.Errorf("failed to decode session response: %w", err)
	}

	a.authToken = sessionResp.Token
	return nil
}

func (a *ArgoInstaller) createApplication(options *InstallOptions) error {
	if options == nil {
		return fmt.Errorf("install options cannot be nil")
	}
	
	app := ArgoApplication{
		APIVersion: "argoproj.io/v1alpha1",
		Kind:       "Application",
		Metadata: ArgoMetadata{
			Name:      options.ApplicationName,
			Namespace: a.ArgoNamespace,
		},
		Spec: ArgoApplicationSpec{
			Project: "default",
			Source: ArgoSource{
				RepoURL:        options.RepoURL,
				Path:           options.Path,
				TargetRevision: options.TargetRevision,
			},
			Destination: ArgoDestination{
				Server:    "https://kubernetes.default.svc",
				Namespace: options.Namespace,
			},
			SyncPolicy: &ArgoSyncPolicy{
				Automated: &ArgoSyncPolicyAutomated{
					Prune:    true,
					SelfHeal: true,
				},
				SyncOptions: []string{"CreateNamespace=true"},
			},
		},
	}

	if app.Spec.Source.Path == "" {
		app.Spec.Source.Path = "."
	}
	if app.Spec.Source.TargetRevision == "" {
		app.Spec.Source.TargetRevision = "HEAD"
	}

	reqBody, err := json.Marshal(app)
	if err != nil {
		return fmt.Errorf("failed to marshal application: %w", err)
	}

	url := fmt.Sprintf("https://%s/api/v1/applications", a.ServerAddress)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create application request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.authToken)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create application: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create application: HTTP %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

func (a *ArgoInstaller) deleteApplication(options *InstallOptions) error {
	if options == nil {
		return fmt.Errorf("install options cannot be nil")
	}
	
	url := fmt.Sprintf("https://%s/api/v1/applications/%s", a.ServerAddress, options.ApplicationName)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.authToken)

	q := req.URL.Query()
	q.Add("cascade", "true")
	req.URL.RawQuery = q.Encode()

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete application: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete application: HTTP %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

func (a *ArgoInstaller) setupPortForward() error {
	podList, err := a.k8sClient.Clientset.CoreV1().Pods(a.ArgoNamespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: labels.Set{"app.kubernetes.io/name": "argocd-server"}.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to list ArgoCD server pods: %w", err)
	}

	if len(podList.Items) == 0 {
		return fmt.Errorf("no ArgoCD server pods found in namespace %s", a.ArgoNamespace)
	}

	pod := podList.Items[0]
	if pod.Status.Phase != corev1.PodRunning {
		return fmt.Errorf("ArgoCD server pod is not running: %s", pod.Status.Phase)
	}

	logger.Info("Setting up port forward to ArgoCD server pod: %s", pod.Name)

	// Create the port forward request URL
	req := a.k8sClient.Clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(a.ArgoNamespace).
		Name(pod.Name).
		SubResource("portforward")

	// Create SPDY transport
	transport, upgrader, err := spdy.RoundTripperFor(a.k8sClient.Config)
	if err != nil {
		return fmt.Errorf("failed to create SPDY transport: %w", err)
	}

	// Set up port forward
	ports := []string{fmt.Sprintf("%d:8080", a.LocalPort)}
	
	a.stopChannel = make(chan struct{}, 1)
	a.readyChannel = make(chan struct{}, 1)
	
	// Create a buffer for output
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	// Create SPDY dialer
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())

	// Create port forwarder
	forwarder, err := portforward.New(
		dialer,
		ports,
		a.stopChannel,
		a.readyChannel,
		out,
		errOut,
	)
	if err != nil {
		return fmt.Errorf("failed to create port forwarder: %w", err)
	}

	// Start port forwarding in a goroutine
	go func() {
		if err := forwarder.ForwardPorts(); err != nil {
			logger.Error("Port forwarding failed: %v", err)
		}
	}()

	// Wait for port forward to be ready
	select {
	case <-a.readyChannel:
		logger.Info("Port forward established successfully")
	case <-time.After(10 * time.Second):
		close(a.stopChannel)
		return fmt.Errorf("timeout waiting for port forward to be ready")
	}

	a.ServerAddress = fmt.Sprintf("localhost:%d", a.LocalPort)
	return nil
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
		return fmt.Errorf("no active connection to ArgoCD")
	}
	return nil
}

func (a *ArgoInstaller) cleanup() {
	a.authToken = ""
	a.ServerAddress = ""
	
	// Terminate port forward if running
	if a.stopChannel != nil {
		logger.Info("Terminating port forward process...")
		close(a.stopChannel)
		a.stopChannel = nil
	}
}
