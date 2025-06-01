package types

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mrgb7/playground/internal/multipass"
)

type Cluster struct {
	Name       string
	Nodes      []*Node
	KubeConfig string
}
type ClusterConfig struct {
	Name               string
	Size               int
	WithCoreComponents bool
	MasterCPUs         int
	MasterMemory       string
	MasterDisk         string
	WorkerCPUs         int
	WorkerMemory       string
	WorkerDisk         string
}

const (
	MaxClusterSize       = 10 // maximum number of nodes allowed in cluster
	MaxClusterNameLength = 63 // maximum length for cluster name (DNS label limit)
	MinClusterSize       = 1  // minimum number of nodes in cluster
	MaxCPUCount          = 32 // maximum number of CPUs per node

)

func NewCluster(name string) *Cluster {
	return &Cluster{
		Name: name,
	}
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
	masterIP, err := cl.GetNodeIP(masterNodeName)
	if err != nil {
		return fmt.Errorf("failed to get master node IP: %w", err)
	}
	res, err := cl.ExecuteShell(masterNodeName, "sudo cat /etc/rancher/k3s/k3s.yaml")
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}
	res = strings.ReplaceAll(res, "127.0.0.1", masterIP)
	c.KubeConfig = res
	return nil
}

func (c *Cluster) GetMasterIP() string {
	masterNodeName := fmt.Sprintf("%s-master", c.Name)
	cl := multipass.NewMultipassClient()
	masterIP, err := cl.GetNodeIP(masterNodeName)
	if err != nil {
		return ""
	}
	return masterIP
}

func (c *Cluster) IsExists() bool {
	cl := multipass.NewMultipassClient()
	_, err := cl.GetClusterInfo(c.Name)
	return err == nil
}

func (c *Cluster) Validate(config ClusterConfig) error {
	if err := validateClusterName(config.Name); err != nil {
		return fmt.Errorf("invalid cluster name: %w", err)
	}

	if err := validateClusterSize(config.Size); err != nil {
		return fmt.Errorf("invalid cluster size: %w", err)
	}

	if err := ValidateCPUCount(config.MasterCPUs, "master"); err != nil {
		return fmt.Errorf("invalid master CPU count: %w", err)
	}

	if err := ValidateMemoryFormat(config.MasterMemory, "master"); err != nil {
		return fmt.Errorf("invalid master memory format: %w", err)
	}

	if err := ValidateDiskFormat(config.MasterDisk, "master"); err != nil {
		return fmt.Errorf("invalid master disk format: %w", err)
	}

	if err := ValidateCPUCount(config.WorkerCPUs, "worker"); err != nil {
		return fmt.Errorf("invalid worker CPU count: %w", err)
	}

	if err := ValidateMemoryFormat(config.WorkerMemory, "worker"); err != nil {
		return fmt.Errorf("invalid worker memory format: %w", err)
	}

	if err := ValidateDiskFormat(config.WorkerDisk, "worker"); err != nil {
		return fmt.Errorf("invalid worker disk format: %w", err)
	}

	return nil
}

func validateClusterName(name string) error {
	if name == "" {
		return fmt.Errorf("cluster name cannot be empty")
	}

	matched, err := regexp.MatchString(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`, name)
	if err != nil {
		return fmt.Errorf("error validating cluster name: %w", err)
	}

	if !matched {
		return fmt.Errorf("cluster name must start and end with alphanumeric characters " +
			"and contain only lowercase letters, numbers, and hyphens")
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

func ValidateCPUCount(cpus int, nodeType string) error {
	if cpus < 1 {
		return fmt.Errorf("%s CPU count must be at least 1", nodeType)
	}
	if cpus > MaxCPUCount {
		return fmt.Errorf("%s CPU count cannot exceed %d", nodeType, MaxCPUCount)
	}
	return nil
}

func ValidateMemoryFormat(memory, nodeType string) error {
	matched, err := regexp.MatchString(`^[0-9]+[GM]$`, memory)
	if err != nil {
		return fmt.Errorf("error validating %s memory format: %w", nodeType, err)
	}
	if !matched {
		return fmt.Errorf("%s memory must be in format like '2G' or '1024M'", nodeType)
	}
	return nil
}

func ValidateDiskFormat(disk, nodeType string) error {
	matched, err := regexp.MatchString(`^[0-9]+[GMT]$`, disk)
	if err != nil {
		return fmt.Errorf("error validating %s disk format: %w", nodeType, err)
	}
	if !matched {
		return fmt.Errorf("%s disk must be in format like '20G', '1024M', or '1T'", nodeType)
	}
	return nil
}
