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
	"gopkg.in/yaml.v3"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
)

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
	K3sInstallTimeout    = 300
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
	
	if len(name) > 63 {
		return fmt.Errorf("cluster name must be 63 characters or less")
	}
	
	return nil
}

func validateClusterSize(size int) error {
	if size < 1 {
		return fmt.Errorf("cluster size must be at least 1")
	}
	
	if size > 10 {
		return fmt.Errorf("cluster size cannot exceed 10 nodes")
	}
	
	return nil
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new cluster",
	Long:  `Create a new cluster with the specified configurations`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := validateClusterName(cCreateName); err != nil {
			logger.Errorf("Invalid cluster name: %v\n", err)
			return
		}
		
		if err := validateClusterSize(cCreateSize); err != nil {
			logger.Errorf("Invalid cluster size: %v\n", err)
			return
		}
		
		var wg sync.WaitGroup
		client := multipass.NewMultipassClient()
		if !client.IsMultipassInstalled() {
			logger.Errorln("Error: Multipass is not installed or not in PATH. Please install Multipass first.")
			return
		}

		type workerError struct {
			nodeName string
			err      error
		}
		workerErrors := make([]workerError, 0)
		var workerErrorsMutex sync.Mutex

		err := client.CreateCluster(cCreateName, cCreateSize, &wg)
		if err != nil {
			logger.Errorln("Failed to create cluster: %v", err)
			return
		}
		
		masterNodeName := fmt.Sprintf("%s-master", cCreateName)
		std, err := client.ExecuteShellWithTimeout(masterNodeName, K3sCreateMasterCmd, K3sInstallTimeout)
		if err != nil || std == "" {
			logger.Errorln("Failed to create k3s on master: %v", err)
			return
		}

		accessToken, err := client.ExecuteShell(masterNodeName, GetAccessTokenCmd)
		if err != nil || accessToken == "" {
			logger.Errorln("Failed to get access token: %v", err)
			return
		}
		accessToken = strings.TrimSpace(accessToken)
		
		kubConfig, err := client.ExecuteShell(masterNodeName, KubeConfigCmd)
		if err != nil || kubConfig == "" {
			logger.Errorln("Failed to get kube config: %v", err)
			return
		}
		
		masterIP, err := client.GetNodeIP(masterNodeName)
		if err != nil || masterIP == "" {
			logger.Errorln("Failed to get master node IP: %v", err)
			return
		}
		
		for i := 0; i < cCreateSize-1; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				nodeName := fmt.Sprintf("%s-worker-%d", cCreateName, i+1)
				_, err = client.ExecuteShellWithTimeout(
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

		if len(workerErrors) > 0 {
			logger.Warnln("Some worker nodes failed to configure properly:")
			for _, we := range workerErrors {
				logger.Errorln("  - %s: %v", we.nodeName, we.err)
			}
			logger.Warnln("Cluster created with %d/%d worker nodes successfully configured", 
				cCreateSize-1-len(workerErrors), cCreateSize-1)
		} else {
			logger.Successln("Successfully created cluster '%s' with %d nodes", cCreateName, cCreateSize)
		}

		logger.Infoln("Attempting to update kubeconfig...")
		if err := createKubeConfigFile(kubConfig); err != nil {
			logger.Errorln("Failed to update kubeconfig: %v", err)
			logger.Warnln("Cluster created successfully, but kubeconfig update failed.")
			logger.Infof("You can manually retrieve the kubeconfig using: playground cluster kubeconfig --name %s\n", cCreateName)
		} else {
			logger.Successln("Successfully updated kubeconfig.")
		}
	},
}

func createKubeConfigFile(kubeConfig string) error {
	var newConfig api.Config
	if err := yaml.Unmarshal([]byte(kubeConfig), &newConfig); err != nil {
		return fmt.Errorf("failed to parse new kubeconfig: %w", err)
	}

	kubeconfigPath := filepath.Join(homedir.HomeDir(), ".kube", "config")
	var existingConfig *api.Config

	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		existingConfig = api.NewConfig()
	} else {
		config, err := clientcmd.LoadFromFile(kubeconfigPath)
		if err != nil {
			return fmt.Errorf("failed to load existing kubeconfig: %w", err)
		}
		existingConfig = config
	}

	for name, cluster := range newConfig.Clusters {
		existingConfig.Clusters[name] = cluster
	}

	for name, authInfo := range newConfig.AuthInfos {
		existingConfig.AuthInfos[name] = authInfo
	}

	for name, context := range newConfig.Contexts {
		existingConfig.Contexts[name] = context
	}

	if newConfig.CurrentContext != "" {
		existingConfig.CurrentContext = newConfig.CurrentContext
	}

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
