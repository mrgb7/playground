package multipass

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/mrgb7/playground/pkg/logger"
)

// Client defines the interface for multipass operations
type Client interface {
	IsMultipassInstalled() bool
	CreateCluster(clusterName string, nodeCount int, wg *sync.WaitGroup) error
	DeleteCluster(clusterName string, wg *sync.WaitGroup) error
	CreateNode(name string, cpus int, memory string, disk string) error
	DeleteNode(name string) error
	PurgeNodes() error
	GetNodeIP(name string) (string, error)
	ExecuteShell(name string, command string) (string, error)
	ExecuteShellWithTimeout(name string, command string, timeoutSeconds int, envs ...string) (string, error)
}

type MultiPassList struct {
	List []MultiPassListItem `json:"list"`
}

type MultiPassListItem struct {
	Name string `json:"name"`
}

type MultiPassInfo struct {
	Errors []interface{}            `json:"errors"`
	Info   map[string]MultiPassNode `json:"info"`
}

type MultiPassNode struct {
	IPv4 []string `json:"ipv4"`
}

type MultipassClient struct {
	BinaryPath string
}

const (
	DefaultMasterCPUs   = 2
	DefaultMasterMemory = "2G"
	DefaultMasterDisk   = "10G"
	DefaultWorkerCPUs   = 1
	DefaultWorkerMemory = "1G"
	DefaultWorkerDisk   = "5G"
)

func NewMultipassClient() *MultipassClient {
	return &MultipassClient{
		BinaryPath: "multipass",
	}
}

func (m *MultipassClient) IsMultipassInstalled() bool {
	// Binary path is controlled, this is a legitimate multipass CLI call
	cmd := exec.Command(m.BinaryPath, "--version") //nolint:gosec
	err := cmd.Run()
	return err == nil
}

func (m *MultipassClient) CreateCluster(clusterName string, nodeCount int, wg *sync.WaitGroup) error {
	masterName := fmt.Sprintf("%s-master", clusterName)
	errChan := make(chan error, nodeCount)

	wg.Add(1)
	go func(name string) {
		defer wg.Done()
		err := m.CreateNode(name, DefaultMasterCPUs, DefaultMasterMemory, DefaultMasterDisk)
		if err != nil {
			logger.Errorf("failed to create master node %s: %v\n", name, err)
			errChan <- fmt.Errorf("failed to create master node %s: %w", name, err)
			return
		}
		logger.Debugln("Master node %s created successfully", name)
	}(masterName)

	for i := 1; i < nodeCount; i++ {
		wg.Add(1)
		go func(workerIndex int) {
			defer wg.Done()
			nodeName := fmt.Sprintf("%s-worker-%d", clusterName, workerIndex)
			err := m.CreateNode(nodeName, DefaultWorkerCPUs, DefaultWorkerMemory, DefaultWorkerDisk)
			if err != nil {
				logger.Errorln("failed to create worker node %s: %v", nodeName, err)
				errChan <- fmt.Errorf("failed to create worker node %s: %w", nodeName, err)
				return
			}
			logger.Debugln("Worker node %s created successfully", nodeName)
		}(i)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	var creationErrors []error
	for err := range errChan {
		if err != nil {
			creationErrors = append(creationErrors, err)
		}
	}

	if len(creationErrors) > 0 {
		logger.Errorln("Error during cluster creation for '%s', attempting cleanup.", clusterName)

		var cleanupWG sync.WaitGroup
		if cleanupErr := m.DeleteCluster(clusterName, &cleanupWG); cleanupErr != nil {
			logger.Errorln("Failed to cleanup cluster %s during error recovery: %v", clusterName, cleanupErr)
		}

		return creationErrors[0]
	}

	logger.Debugln("Cluster %s created successfully with %d total instances.", clusterName, nodeCount)
	return nil
}

func (m *MultipassClient) DeleteCluster(clusterName string, wg *sync.WaitGroup) error {
	var list MultiPassList
	// Binary path is controlled, this is a legitimate multipass CLI call
	cmd := exec.Command(m.BinaryPath, "list", "--format", "json") //nolint:gosec
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to list instances: %s - %w", stderr.String(), err)
	}
	if err := json.Unmarshal(stdout.Bytes(), &list); err != nil {
		return fmt.Errorf("failed to parse JSON output: %w", err)
	}

	var instancesToDelete []string
	for _, instance := range list.List {
		if strings.HasPrefix(instance.Name, clusterName) {
			instancesToDelete = append(instancesToDelete, instance.Name)
		}
	}

	errChan := make(chan error, len(instancesToDelete))

	for _, instanceName := range instancesToDelete {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := m.DeleteNode(name); err != nil {
				errChan <- fmt.Errorf("failed to delete node %s: %w", name, err)
				return
			}
			logger.Debugf("Successfully deleted node: %s", name)
		}(instanceName)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	var errors []error
	for err := range errChan {
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		var errMessages []string
		for _, err := range errors {
			errMessages = append(errMessages, err.Error())
		}
		return fmt.Errorf("multiple deletion errors: %s", strings.Join(errMessages, "; "))
	}

	return nil
}

func (m *MultipassClient) CreateNode(name string, cpus int, memory string, disk string) error {
	args := []string{
		"launch",
		"--name", name,
		"--cpus", fmt.Sprintf("%d", cpus),
		"--memory", memory,
		"--disk", disk,
	}

	logger.Debugln("Creating node: %s with %d CPUs, %s memory, %s disk", name, cpus, memory, disk)
	// Binary path is controlled, this is a legitimate multipass CLI call
	cmd := exec.Command(m.BinaryPath, args...) //nolint:gosec
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create node '%s': %s - %w", name, stderr.String(), err)
	}

	return nil
}

func (m *MultipassClient) DeleteNode(name string) error {
	// Binary path is controlled, this is a legitimate multipass CLI call
	cmd := exec.Command(m.BinaryPath, "delete", name) //nolint:gosec
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete node '%s': %s - %w", name, stderr.String(), err)
	}

	logger.Debugln("Successfully deleted node '%s'", name)
	return nil
}

func (m *MultipassClient) PurgeNodes() error {
	logger.Infoln("Purging deleted nodes")
	// Binary path is controlled, this is a legitimate multipass CLI call
	cmd := exec.Command(m.BinaryPath, "purge") //nolint:gosec
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to purge deleted instances: %s - %w", stderr.String(), err)
	}

	logger.Successln("Successfully purged deleted nodes")
	return nil
}

func (m *MultipassClient) GetNodeIP(name string) (string, error) {
	// Binary path is controlled, this is a legitimate multipass CLI call
	cmd := exec.Command(m.BinaryPath, "info", name, "--format", "json") //nolint:gosec
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get IP address for node '%s': %s - %w", name, stderr.String(), err)
	}

	var data MultiPassInfo
	if err := json.Unmarshal(stdout.Bytes(), &data); err != nil {
		return "", fmt.Errorf("failed to parse JSON output: %w", err)
	}

	nodeInfo, exists := data.Info[name]
	if !exists {
		return "", fmt.Errorf("node '%s' not found in multipass info", name)
	}

	if len(nodeInfo.IPv4) == 0 {
		return "", fmt.Errorf("no IPv4 addresses found for node '%s'", name)
	}

	ip := nodeInfo.IPv4[0]
	return ip, nil
}

func (m *MultipassClient) ExecuteShell(name string, command string) (string, error) {
	return m.ExecuteShellWithTimeout(name, command, 0) // No timeout by default
}

func (m *MultipassClient) ExecuteShellWithTimeout(name string, command string, timeoutSeconds int,
	envs ...string) (string, error) {
	ctx := context.Background()
	var cancel context.CancelFunc

	if timeoutSeconds > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
		defer cancel()
	}

	// Binary path is controlled, this is a legitimate multipass CLI call
	cmd := exec.CommandContext(ctx, m.BinaryPath, "exec", name, "--", "bash", "-c", command) //nolint:gosec
	cmd.Env = append(os.Environ(), envs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		logger.Errorln("Failed to execute command on node '%s': %v", name, err)
		if ctx.Err() == context.DeadlineExceeded {
			return stdout.String(), fmt.Errorf("command timed out after %d seconds", timeoutSeconds)
		}

		errMsg := fmt.Sprintf("Failed to execute shell command on node '%s': %s", name, stderr.String())
		logger.Errorln(errMsg)
		return "", fmt.Errorf("failed to execute shell command on node '%s': %s - %w", name, stderr.String(), err)
	}

	return stdout.String(), nil
}
