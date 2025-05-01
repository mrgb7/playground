package types

import (
	"fmt"
	"strings"

	"github.com/mohamedragab2024/playground/internal/multipass"
)

type Cluster struct {
	Name       string
	Nodes      []*Node
	KubeConfig string
}

func Get(clusterName string, load *bool) *Cluster {
	return nil
}

func (c *Cluster) GetMaster() *Node {
	for _, node := range c.Nodes {
		if strings.HasSuffix(node.Name, "master") {
			return node
		}
	}
	return nil
}

func (c *Cluster) GetWorkers() []*Node {
	workers := []*Node{}
	for _, node := range c.Nodes {
		if !strings.HasSuffix(node.Name, "master") {
			workers = append(workers, node)
		}
	}
	return workers
}

func (c *Cluster) SetKubeConfig() error {
	masterNodeName := fmt.Sprintf("%s-master", c.Name)
	cl := multipass.NewMultipassClient()
	// Get the master node IP
	masterIP, err := cl.GetNodeIP(masterNodeName)
	if err != nil {
		return fmt.Errorf("failed to get master node IP: %w", err)
	}
	res, err := cl.ExcuteShell(masterNodeName, "sudo cat /etc/rancher/k3s/k3s.yaml")
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}
	// Replace the server address in the kubeconfig with the master node masterIP
	res = strings.ReplaceAll(res, "127.0.0.1", masterIP)
	c.KubeConfig = res
	return nil
}
