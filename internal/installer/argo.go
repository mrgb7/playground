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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type ArgoInstaller struct {
	KubeConfig     string
	ClusterName    string
	ArgoNamespace  string
	ArgoServerPort int
	LocalPort      int
	ServerAddress  string
	k8sClient      *k8s.K8sClient
	httpClient     *http.Client
	authToken      string
	stopChannel    chan struct{}
	readyChannel   chan struct{}
}

type ArgoApplication struct {
	APIVersion string              `json:"apiVersion"`
	Kind       string              `json:"kind"`
	Metadata   ArgoMetadata        `json:"metadata"`
	Spec       ArgoApplicationSpec `json:"spec"`
}

type ArgoMetadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type ArgoApplicationSpec struct {
	Project     string          `json:"project"`
	Source      ArgoSource      `json:"source"`
	Destination ArgoDestination `json:"destination"`
	SyncPolicy  *ArgoSyncPolicy `json:"syncPolicy,omitempty"`
}

type ArgoSource struct {
	RepoURL        string  `json:"repoURL"`
	Path           string  `json:"path"`
	TargetRevision string  `json:"targetRevision"`
	Chart          *string `json:"chart,omitempty"` // Optional, used for Helm charts
	Helm           struct {
		ReleaseName  string `json:"releaseName,omitempty"`
		ValuesObject map[string]interface{}
	} `json:"helm,omitempty"` // Optional, used for Helm charts
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
			// This is for local development with port forwarding to ArgoCD
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
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

	logger.Infoln("Starting ArgoCD application installation...")

	if err := a.connectToArgoCD(); err != nil {
		return fmt.Errorf("failed to connect to ArgoCD: %w", err)
	}
	defer a.cleanup()

	if err := a.createApplication(options); err != nil {
		return fmt.Errorf("failed to create ArgoCD application: %w", err)
	}

	logger.Infoln("Successfully created ArgoCD application: %s", options.ApplicationName)
	return nil
}

func (a *ArgoInstaller) UnInstall(options *InstallOptions) error {
	if options == nil {
		return fmt.Errorf("install options cannot be nil")
	}

	logger.Infoln("Starting ArgoCD application uninstallation...")

	if err := a.connectToArgoCD(); err != nil {
		return fmt.Errorf("failed to connect to ArgoCD: %w", err)
	}
	defer a.cleanup()

	if err := a.deleteApplication(options); err != nil {
		return fmt.Errorf("failed to delete ArgoCD application: %w", err)
	}

	if options.Plugin != nil && options.Plugin.OwnsNamespace() {
		namespaceManager := NewNamespaceManager(a.KubeConfig)
		if err := namespaceManager.DeleteNamespace(options.Namespace); err != nil {
			logger.Warnf("Warning: Failed to cleanup namespace: %v", err)
		}
	}

	logger.Infoln("Successfully deleted ArgoCD application: %s", options.ApplicationName)
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

	// Wait a bit longer for the port forward to be fully established
	logger.Infoln("Waiting for port forward to stabilize...")
	time.Sleep(5 * time.Second)

	// Retry authentication with backoff
	var authErr error
	for i := 0; i < 3; i++ {
		authErr = a.authenticate(password)
		if authErr == nil {
			return nil
		}
		logger.Warnln("Authentication attempt %d failed: %v, retrying...", i+1, authErr)
		time.Sleep(time.Duration(i+1) * 2 * time.Second)
	}

	return fmt.Errorf("failed to authenticate after 3 attempts: %w", authErr)
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

	url := fmt.Sprintf("http://%s/api/v1/session", a.ServerAddress)
	req, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create session request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Debugln("Failed to close response body: %v", err)
		}
	}()

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
				TargetRevision: options.Version,
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
	if options.ChartName != nil {
		app.Spec.Source.Chart = options.ChartName
		app.Spec.Source.Helm.ReleaseName = options.ApplicationName
		app.Spec.Source.Helm.ValuesObject = options.Values
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

	url := fmt.Sprintf("http://%s/api/v1/applications", a.ServerAddress)
	req, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create application request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.authToken)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create application: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Debugln("Failed to close response body: %v", err)
		}
	}()

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

	url := fmt.Sprintf("http://%s/api/v1/applications/%s", a.ServerAddress, options.ApplicationName)
	req, err := http.NewRequestWithContext(context.Background(), "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.authToken)
	req.Header.Set("Content-Type", "application/json")

	q := req.URL.Query()
	q.Add("cascade", "true")
	req.URL.RawQuery = q.Encode()

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete application: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Debugln("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent &&
		resp.StatusCode != http.StatusNotFound {
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

	logger.Infoln("Setting up port forward to ArgoCD server pod: %s", pod.Name)
	req := a.k8sClient.Clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(a.ArgoNamespace).
		Name(pod.Name).
		SubResource("portforward")
	transport, upgrader, err := spdy.RoundTripperFor(a.k8sClient.Config)
	if err != nil {
		return fmt.Errorf("failed to create SPDY transport: %w", err)
	}
	ports := []string{fmt.Sprintf("%d:8080", a.LocalPort)}

	a.stopChannel = make(chan struct{}, 1)
	a.readyChannel = make(chan struct{}, 1)
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())
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

	errChan := make(chan error, 1)
	go func() {
		if err := forwarder.ForwardPorts(); err != nil {
			logger.Errorln("Port forwarding failed: %v", err)
			errChan <- err
		}
	}()

	select {
	case <-a.readyChannel:
		logger.Infoln("Port forward established successfully")
	case err := <-errChan:
		close(a.stopChannel)
		return fmt.Errorf("port forwarding failed: %w", err)
	case <-time.After(15 * time.Second):
		close(a.stopChannel)
		return fmt.Errorf("timeout waiting for port forward to be ready")
	}

	a.ServerAddress = fmt.Sprintf("localhost:%d", a.LocalPort)
	return nil
}

func (a *ArgoInstaller) GetAdminPassword() (string, error) {
	secret, err := a.k8sClient.Clientset.CoreV1().Secrets(a.ArgoNamespace).Get(
		context.Background(), "argocd-initial-admin-secret", metav1.GetOptions{})
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
	if a.stopChannel != nil {
		logger.Infoln("Terminating port forward process...")
		close(a.stopChannel)
		a.stopChannel = nil
	}
}
