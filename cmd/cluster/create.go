package cluster

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/mrgb7/playground/internal/multipass"
	"github.com/mrgb7/playground/pkg/logger"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
)

// ClusterConfig holds the configuration for cluster creation
type ClusterConfig struct {
	Name               string
	Size               int
	WithCoreComponents bool
}

// workerError represents an error that occurred while configuring a worker node
type workerError struct {
	nodeName string
	err      error
}

var (
	cCreateName        string
	cCreateSize        int
	withCoreComponents bool
)

const (
	K3sCreateMasterCmd   = `curl -sfL https://get.k3s.io | sh -s - --disable=servicelb --disable=traefik`
	GetAccessTokenCmd    = `sudo cat /var/lib/rancher/k3s/server/node-token`
	K3sCreateWorkerCmd   = `curl -sfL https://get.k3s.io | K3S_URL=https://%s:6443 K3S_TOKEN=%s  sh -`
	KubeConfigCmd        = `sudo cat /etc/rancher/k3s/k3s.yaml`
	K3sInstallTimeout    = 300 // seconds - timeout for K3s installation
	MaxClusterSize       = 10  // maximum number of nodes allowed in cluster
	MaxClusterNameLength = 63  // maximum length for cluster name (DNS label limit)
	MinClusterSize       = 1   // minimum number of nodes in cluster
)

func validateClusterName(name string) error {
	if name == "" {
		return fmt.Errorf("cluster name cannot be empty")
	}

	matched, err := regexp.MatchString(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`, name)
	if err != nil {
		return fmt.Errorf("error validating cluster name: %w", err)
	}

	if !matched {
		return fmt.Errorf("cluster name must start and end with alphanumeric characters and contain only lowercase letters, numbers, and hyphens")
	}

	if len(name) > MaxClusterNameLength {
		return fmt.Errorf("cluster name must be %d characters or less", MaxClusterNameLength)
	}

	return nil
}

func validateClusterSize(size int) error {
	if size < MinClusterSize {
		return fmt.Errorf("cluster size must be at least %d", MinClusterSize)
	}

	if size > MaxClusterSize {
		return fmt.Errorf("cluster size cannot exceed %d nodes", MaxClusterSize)
	}

	return nil
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new cluster",
	Long:  `Create a new cluster with the specified configurations`,
	Run: func(cmd *cobra.Command, args []string) {
		config := &ClusterConfig{
			Name:               cCreateName,
			Size:               cCreateSize,
			WithCoreComponents: withCoreComponents,
		}

		if err := createCluster(config); err != nil {
			logger.Errorln("Failed to create cluster: %v", err)
			return
		}
	},
}

func createCluster(config *ClusterConfig) error {
	if err := validateClusterName(config.Name); err != nil {
		return fmt.Errorf("invalid cluster name: %w", err)
	}

	if err := validateClusterSize(config.Size); err != nil {
		return fmt.Errorf("invalid cluster size: %w", err)
	}

	client := multipass.NewMultipassClient()
	if !client.IsMultipassInstalled() {
		return fmt.Errorf("multipass is not installed or not in PATH")
	}

	return executeClusterCreation(client, config)
}

func executeClusterCreation(client multipass.Client, config *ClusterConfig) error {
	var wg sync.WaitGroup

	if err := client.CreateCluster(config.Name, config.Size, &wg); err != nil {
		return fmt.Errorf("failed to create cluster: %w", err)
	}

	masterNodeName := fmt.Sprintf("%s-master", config.Name)

	// Install K3s on master node
	if err := installMasterNode(client, masterNodeName); err != nil {
		return fmt.Errorf("failed to install K3s on master: %w", err)
	}

	// Get access token and master IP
	accessToken, masterIP, err := getMasterCredentials(client, masterNodeName)
	if err != nil {
		return fmt.Errorf("failed to get master credentials: %w", err)
	}

	// Configure worker nodes
	workerErrors := configureWorkerNodes(client, config, masterIP, accessToken)

	// Report results
	reportClusterCreationResults(config, workerErrors)

	// Update kubeconfig
	return updateKubeConfig(client, masterNodeName, config.Name)
}

func installMasterNode(client multipass.Client, masterNodeName string) error {
	std, err := client.ExecuteShellWithTimeout(masterNodeName, K3sCreateMasterCmd, K3sInstallTimeout)
	if err != nil || std == "" {
		return fmt.Errorf("failed to create k3s on master: %w", err)
	}
	return nil
}

func getMasterCredentials(client multipass.Client, masterNodeName string) (string, string, error) {
	accessToken, err := client.ExecuteShell(masterNodeName, GetAccessTokenCmd)
	if err != nil || accessToken == "" {
		return "", "", fmt.Errorf("failed to get access token: %w", err)
	}
	accessToken = strings.TrimSpace(accessToken)

	masterIP, err := client.GetNodeIP(masterNodeName)
	if err != nil || masterIP == "" {
		return "", "", fmt.Errorf("failed to get master node IP: %w", err)
	}

	return accessToken, masterIP, nil
}

func configureWorkerNodes(client multipass.Client, config *ClusterConfig, masterIP, accessToken string) []workerError {
	workerErrors := make([]workerError, 0)
	var workerErrorsMutex sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < config.Size-1; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			nodeName := fmt.Sprintf("%s-worker-%d", config.Name, i+1)
			_, err := client.ExecuteShellWithTimeout(
				nodeName,
				fmt.Sprintf(K3sCreateWorkerCmd, masterIP, accessToken),
				K3sInstallTimeout,
			)
			if err != nil {
				workerErrorsMutex.Lock()
				workerErrors = append(workerErrors, workerError{
					nodeName: nodeName,
					err:      err,
				})
				workerErrorsMutex.Unlock()
				logger.Errorln("Failed to install K3S on worker node %s: %v", nodeName, err)
			} else {
				logger.Successf("Successfully configured worker node: %s\n", nodeName)
			}
		}(i)
	}
	wg.Wait()

	return workerErrors
}

func reportClusterCreationResults(config *ClusterConfig, workerErrors []workerError) {
	if len(workerErrors) > 0 {
		logger.Warnln("Some worker nodes failed to configure properly:")
		for _, we := range workerErrors {
			logger.Errorln("  - %s: %v", we.nodeName, we.err)
		}
		logger.Warnln("Cluster created with %d/%d worker nodes successfully configured",
			config.Size-1-len(workerErrors), config.Size-1)
	} else {
		logger.Successln("Successfully created cluster '%s' with %d nodes", config.Name, config.Size)
	}
}

func updateKubeConfig(client multipass.Client, masterNodeName, clusterName string) error {
	logger.Infoln("Attempting to update kubeconfig...")

	kubConfig, err := client.ExecuteShell(masterNodeName, KubeConfigCmd)
	if err != nil || kubConfig == "" {
		return fmt.Errorf("failed to get kube config: %w", err)
	}

	// Get master IP to replace 127.0.0.1 in kubeconfig
	masterIP, err := client.GetNodeIP(masterNodeName)
	if err != nil {
		return fmt.Errorf("failed to get master IP: %w", err)
	}

	// Replace localhost with master IP
	kubConfig = strings.ReplaceAll(kubConfig, "127.0.0.1", masterIP)

	if err := createKubeConfigFile(kubConfig, clusterName); err != nil {
		logger.Errorln("Failed to update kubeconfig: %v", err)
		logger.Warnln("Cluster created successfully, but kubeconfig update failed.")
		logger.Infof("You can manually retrieve the kubeconfig using: playground cluster kubeconfig --name %s\n", clusterName)
		return err
	}

	logger.Successln("Successfully updated kubeconfig.")
	return nil
}

func createKubeConfigFile(kubeConfig, clusterName string) error {
	// Use client-go to properly parse the K3s kubeconfig format
	newConfig, err := clientcmd.Load([]byte(kubeConfig))
	if err != nil {
		return fmt.Errorf("failed to parse new kubeconfig: %w", err)
	}

	// Update context and cluster names to include cluster name
	contextName := fmt.Sprintf("%s-context", clusterName)
	clusterKey := fmt.Sprintf("%s-cluster", clusterName)
	userKey := fmt.Sprintf("%s-user", clusterName)

	// Rename the default entries to use cluster-specific names
	if cluster, exists := newConfig.Clusters["default"]; exists {
		delete(newConfig.Clusters, "default")
		newConfig.Clusters[clusterKey] = cluster
	}

	if authInfo, exists := newConfig.AuthInfos["default"]; exists {
		delete(newConfig.AuthInfos, "default")
		newConfig.AuthInfos[userKey] = authInfo
	}

	if context, exists := newConfig.Contexts["default"]; exists {
		delete(newConfig.Contexts, "default")
		context.Cluster = clusterKey
		context.AuthInfo = userKey
		newConfig.Contexts[contextName] = context
	}

	// Set current context to the new cluster
	newConfig.CurrentContext = contextName

	kubeconfigPath := filepath.Join(homedir.HomeDir(), ".kube", "config")
	var existingConfig *api.Config

	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		existingConfig = api.NewConfig()
	} else {
		existingConfig, err = clientcmd.LoadFromFile(kubeconfigPath)
		if err != nil {
			return fmt.Errorf("failed to load existing kubeconfig: %w", err)
		}
	}

	// Merge configurations
	for name, cluster := range newConfig.Clusters {
		existingConfig.Clusters[name] = cluster
	}

	for name, authInfo := range newConfig.AuthInfos {
		existingConfig.AuthInfos[name] = authInfo
	}

	for name, context := range newConfig.Contexts {
		existingConfig.Contexts[name] = context
	}

	// Set current context to the new cluster
	existingConfig.CurrentContext = contextName

	if err := clientcmd.WriteToFile(*existingConfig, kubeconfigPath); err != nil {
		return fmt.Errorf("failed to write merged kubeconfig: %w", err)
	}

	return nil
}

func init() {
	createCmd.Flags().StringVarP(&cCreateName, "name", "n", "", "Name for the cluster (required)")
	createCmd.Flags().IntVarP(&cCreateSize, "size", "s", 1, "Number of nodes in the cluster")
	createCmd.Flags().BoolVarP(&withCoreComponents, "with-core-component", "c", false, "Install core components (nginx,cert-manager)")
	createCmd.MarkFlagRequired("name")
}
