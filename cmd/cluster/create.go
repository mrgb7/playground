package cluster

import (
	"fmt"

	"github.com/mohamedragab2024/playground/internal/multipass"
	"github.com/mohamedragab2024/playground/pkg/logger"
	"github.com/spf13/cobra"
)

var (
	cCreateName           string
	cCreateSize           int
	K3S_CREATE_MASTER_CMD = `curl -sfL https://get.k3s.io | sh -s - --disable=servicelb --disable=traefik`
	GET_ACCESS_TOKEN_CMD  = `sudo cat /var/lib/rancher/k3s/server/node-token`
	K3S_CREATE_WORKER_CMD = `curl -sfL https://get.k3s.io | K3S_URL=https://%s:6443 K3S_TOKEN="%s" sh -`
	KUBE_CONFIG_CMD       = `sudo cat /etc/rancher/k3s/k3s.yaml`
	K3S_INSTALL_TIMEOUT   = 300 // 5 minutes timeout for K3S installation
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new cluster",
	Long:  `Create a new cluster with the specified configurations`,
	Run: func(cmd *cobra.Command, args []string) {
		client := multipass.NewMultipassClient()
		if client.IsMultipassInstalled() {
		}
		if !client.IsMultipassInstalled() {
			logger.Errorln("Error: Multipass is not installed or not in PATH. Please install Multipass first.")
			return
		}

		err := client.CreateCluster(cCreateName, cCreateSize)
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
		for i := 0; i < (cCreateSize - 1); i++ {
			nodeName := fmt.Sprintf("%s-worker-%d", cCreateName, i+1)
			command := fmt.Sprintf(K3S_CREATE_WORKER_CMD, masterIP, accessToken)
			logger.Infoln("Executing command: %s", command)
			_, err = client.ExcuteShellWithTimeout(
				nodeName,
				command,
				K3S_INSTALL_TIMEOUT,
			)
			if err != nil {
				logger.Errorln("Failed to install K3S on worker node %s: %v", nodeName, err)
				return
			}

		}

		logger.Successln("Successfully created cluster '%s' with %d nodes", cCreateName, cCreateSize)
		logger.Infoln("Kube config: %s", kubConfig)
	},
}

func init() {
	ClusterCmd.AddCommand(createCmd)

	createCmd.Flags().StringVarP(&cCreateName, "name", "n", "", "Name for the cluster (required)")
	createCmd.Flags().IntVarP(&cCreateSize, "size", "s", 1, "Number of nodes in the cluster")

	createCmd.MarkFlagRequired("name")
}
