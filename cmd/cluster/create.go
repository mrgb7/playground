package cluster

import (
	"fmt"
	"os"
	"path/filepath"
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
	K3S_CREATE_MASTER_CMD = `curl -sfL https://get.k3s.io | sh -s - --disable=servicelb --disable=traefik`
	GET_ACCESS_TOKEN_CMD  = `sudo cat /var/lib/rancher/k3s/server/node-token`
	K3S_CREATE_WORKER_CMD = `curl -sfL https://get.k3s.io | K3S_URL=https://%s:6443 K3S_TOKEN=%s  sh -`
	KUBE_CONFIG_CMD       = `sudo cat /etc/rancher/k3s/k3s.yaml`
	K3S_INSTALL_TIMEOUT   = 300 // 5 minutes timeout for K3S installation
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new cluster",
	Long:  `Create a new cluster with the specified configurations`,
	Run: func(cmd *cobra.Command, args []string) {
		var wg sync.WaitGroup
		client := multipass.NewMultipassClient()
		if !client.IsMultipassInstalled() {
			logger.Errorln("Error: Multipass is not installed or not in PATH. Please install Multipass first.")
			return
		}

		err := client.CreateCluster(cCreateName, cCreateSize, &wg)
		if err != nil {
			logger.Errorln("Failed to create cluster: %v", err)
			return
		}
		masterNodeName := fmt.Sprintf("%s-master", cCreateName)
		std, err := client.ExcuteShellWithTimeout(masterNodeName, K3S_CREATE_MASTER_CMD, K3S_INSTALL_TIMEOUT)
		if err != nil || std == "" {
			logger.Errorln("Failed to create k3s on master: %v", err)
			return
		}

		accessToken, err := client.ExcuteShell(masterNodeName, GET_ACCESS_TOKEN_CMD)
		if err != nil || accessToken == "" {
			logger.Errorln("Failed to get access token: %v", err)
			return
		}
		accessToken = strings.TrimSpace(accessToken) // Trim whitespace from accessToken
		kubConfig, err := client.ExcuteShell(masterNodeName, KUBE_CONFIG_CMD)
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
				nodeName := fmt.Sprintf("%s-worker-%d", cCreateName, i+1)
				_, err = client.ExcuteShellWithTimeout(
					nodeName,
					fmt.Sprintf(K3S_CREATE_WORKER_CMD, masterIP, accessToken),
					K3S_INSTALL_TIMEOUT,
				)
				if err != nil {
					logger.Errorln("Failed to install K3S on worker node %s: %v", nodeName, err)
					return
				}
				wg.Done()
			}(i)

		}
		wg.Wait()

		logger.Successln("Successfully created cluster '%s' with %d nodes", cCreateName, cCreateSize)

		logger.Infoln("Attempting to update kubeconfig...")
		if err := createkubeConfigFile(kubConfig); err != nil {
			logger.Errorln("Failed to update kubeconfig: %v", err)
			// Not returning an error here as the cluster is created,
			// but kubeconfig update failed. User can retrieve it manually.
		} else {
			logger.Successln("Successfully updated kubeconfig.")
		}
	},
}

func createkubeConfigFile(kubeConfig string) error {
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
	ClusterCmd.AddCommand(createCmd)
	createCmd.Flags().StringVarP(&cCreateName, "name", "n", "", "Name for the cluster (required)")
	createCmd.Flags().IntVarP(&cCreateSize, "size", "s", 1, "Number of nodes in the cluster")
	createCmd.Flags().BoolVarP(&withCoreComponents, "with-core-component", "c", false, "Install core components (nginx,cert-manager)")
	createCmd.MarkFlagRequired("name")
}
