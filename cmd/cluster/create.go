package cluster

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mrgb7/playground/internal/multipass"
	"github.com/mrgb7/playground/pkg/logger"
	"github.com/mrgb7/playground/types"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
)

// ClusterConfig holds the configuration for cluster creation

// workerError represents an error that occurred while configuring a worker node
type workerError struct {
	nodeName string
	err      error
}

var (
	cCreateName        string
	cCreateSize        int
	withCoreComponents bool
	masterCPUs         int
	masterMemory       string
	masterDisk         string
	workerCPUs         int
	workerMemory       string
	workerDisk         string
)

const (
	K3sCreateMasterCmd = `curl -sfL https://get.k3s.io | sh -s - --disable=servicelb --disable=traefik`
	GetAccessTokenCmd  = `sudo cat /var/lib/rancher/k3s/server/node-token` //nolint:gosec
	K3sCreateWorkerCmd = `curl -sfL https://get.k3s.io | K3S_URL=https://%s:6443 K3S_TOKEN=%s  sh -`
	KubeConfigCmd      = `sudo cat /etc/rancher/k3s/k3s.yaml`
	K3sInstallTimeout  = 300 // seconds - timeout for K3s installation
	DefaultMasterCPUs  = 2   // default number of CPUs for master node
	DefaultWorkerCPUs  = 2   // default number of CPUs for worker nodes

)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new cluster",
	Long:  `Create a new cluster with the specified configurations`,
	Run: func(cmd *cobra.Command, args []string) {
		config := &types.ClusterConfig{
			Name:               cCreateName,
			Size:               cCreateSize,
			WithCoreComponents: withCoreComponents,
			MasterCPUs:         masterCPUs,
			MasterMemory:       masterMemory,
			MasterDisk:         masterDisk,
			WorkerCPUs:         workerCPUs,
			WorkerMemory:       workerMemory,
			WorkerDisk:         workerDisk,
		}

		if err := createCluster(config); err != nil {
			logger.Errorf("Failed to create cluster: %v", err)
			return
		}
	},
}

func createCluster(config *types.ClusterConfig) error {
	client := multipass.NewMultipassClient()

	if !client.IsMultipassInstalled() {
		return fmt.Errorf("multipass is not installed or not in PATH")
	}

	cl := types.NewCluster(config.Name)

	err := cl.Validate(*config)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	if cl.IsExists() {
		return fmt.Errorf("cluster '%s' already exists", config.Name)
	}

	return executeClusterCreation(client, config)
}

func executeClusterCreation(client multipass.Client, config *types.ClusterConfig) error {
	var wg sync.WaitGroup

	if err := client.CreateCluster(
		config.Name, config.Size, config.MasterCPUs, config.MasterMemory, config.MasterDisk,
		config.WorkerCPUs, config.WorkerMemory, config.WorkerDisk, &wg,
	); err != nil {
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

func configureWorkerNodes(client multipass.Client, config *types.ClusterConfig, masterIP, accessToken string) []workerError {
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

func reportClusterCreationResults(config *types.ClusterConfig, workerErrors []workerError) {
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
	createCmd.Flags().BoolVarP(&withCoreComponents, "with-core-component", "c", false,
		"Install core components (nginx,cert-manager)")
	createCmd.Flags().IntVarP(&masterCPUs, "master-cpus", "m", DefaultMasterCPUs, "Number of CPUs for the master node")
	createCmd.Flags().StringVarP(&masterMemory, "master-memory", "M", "2G", "Memory for the master node")
	createCmd.Flags().StringVarP(&masterDisk, "master-disk", "D", "20G", "Disk for the master node")
	createCmd.Flags().IntVarP(&workerCPUs, "worker-cpus", "w", DefaultWorkerCPUs, "Number of CPUs for each worker node")
	createCmd.Flags().StringVarP(&workerMemory, "worker-memory", "W", "2G", "Memory for each worker node")
	createCmd.Flags().StringVarP(&workerDisk, "worker-disk", "d", "20G", "Disk for each worker node")
	if err := createCmd.MarkFlagRequired("name"); err != nil {
		logger.Errorln("Failed to mark name flag as required: %v", err)
	}
}
