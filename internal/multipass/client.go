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

type MultiPassList struct {
	List []MultiPassListItem `json:"list"`
}

type MultiPassListItem struct {
	Name string `json:"name"`
}
type MultiPassInfo struct {
	Errors []interface{} `json:"errors"`
	Info   struct {
		PlaygroundMaster struct {
			IPv4 []string `json:"ipv4"`
		} `json:"playground-master"`
	} `json:"info"`
}
type MultipassClient struct {
	BinaryPath string
}

func NewMultipassClient() *MultipassClient {
	return &MultipassClient{
		BinaryPath: "multipass",
	}
}

func (m *MultipassClient) IsMultipassInstalled() bool {
	cmd := exec.Command(m.BinaryPath, "--version")
	err := cmd.Run()
	return err == nil
}

func (m *MultipassClient) CreateCluster(clusterName string, nodeCount int, wg *sync.WaitGroup) error {
	masterName := fmt.Sprintf("%s-master", clusterName)
	errChan := make(chan error, nodeCount)
	
	// Create master node
	wg.Add(1)
	go func(name string) {
		defer wg.Done()
		err := m.CreateNode(name, 2, "2G", "10G")
		if err != nil {
			logger.Error("failed to create master node %s: %v", name, err)
			errChan <- fmt.Errorf("failed to create master node %s: %w", name, err)
			return
		}
		logger.Debugln("Master node %s created successfully", name)
	}(masterName)
	
	// Create worker nodes
	for i := 1; i < nodeCount; i++ {
		wg.Add(1)
		go func(workerIndex int) {
			defer wg.Done()
			nodeName := fmt.Sprintf("%s-worker-%d", clusterName, workerIndex)
			err := m.CreateNode(nodeName, 1, "1G", "5G")
			if err != nil {
				logger.Errorln("failed to create worker node %s: %v", nodeName, err)
				errChan <- fmt.Errorf("failed to create worker node %s: %w", nodeName, err)
				return
			}
			logger.Debugln("Worker node %s created successfully", nodeName)
		}(i)
	}
	
	// Wait for all nodes to be created and close error channel
	go func() {
		wg.Wait()
		close(errChan)
	}()
	
	// Check for any creation errors
	var creationErrors []error
	for err := range errChan {
		if err != nil {
			creationErrors = append(creationErrors, err)
		}
	}
	
	// If there were errors, attempt cleanup and return the first error
	if len(creationErrors) > 0 {
		logger.Errorln("Error during cluster creation for '%s', attempting cleanup.", clusterName)
		
		// Create a new WaitGroup for cleanup operations
		var cleanupWG sync.WaitGroup
		if cleanupErr := m.DeleteCluster(clusterName, &cleanupWG); cleanupErr != nil {
			logger.Errorln("Failed to cleanup cluster %s during error recovery: %v", clusterName, cleanupErr)
		}
		
		// Return the first creation error
		return creationErrors[0]
	}
	
	logger.Debugln("Cluster %s created successfully with %d total instances.", clusterName, nodeCount)
	return nil
}

func (m *MultipassClient) DeleteCluster(clusterName string, wg *sync.WaitGroup) error {
	var list MultiPassList
	cmd := exec.Command(m.BinaryPath, "list", "--format", "json")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		errMsg := fmt.Sprintf("Failed to list instances: %s", stderr.String())
		logger.Errorln(errMsg)
		return fmt.Errorf("failed to list instances: %s - %w", stderr.String(), err)
	}
	if err := json.Unmarshal(stdout.Bytes(), &list); err != nil {
		errMsg := fmt.Sprintf("Failed to parse JSON output: %s", err)
		logger.Errorln(errMsg)
		return fmt.Errorf("failed to parse JSON output: %s - %w", err, err)
	}
	
	// Collect instances to delete first
	var instancesToDelete []string
	for _, instance := range list.List {
		if strings.HasPrefix(instance.Name, clusterName) {
			instancesToDelete = append(instancesToDelete, instance.Name)
		}
	}
	
	// Create error channel for collecting deletion errors
	errChan := make(chan error, len(instancesToDelete))
	
	// Start deletion goroutines
	for _, instanceName := range instancesToDelete {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := m.DeleteNode(name); err != nil {
				errChan <- fmt.Errorf("failed to delete node %s: %w", name, err)
				return
			}
			logger.Debugln("Successfully deleted node: %s", name)
		}(instanceName)
	}
	
	// Wait for all deletions to complete
	go func() {
		wg.Wait()
		close(errChan)
	}()
	
	// Collect any errors
	var errors []error
	for err := range errChan {
		if err != nil {
			errors = append(errors, err)
		}
	}
	
	// Return combined errors if any occurred
	if len(errors) > 0 {
		var errorMessages []string
		for _, err := range errors {
			errorMessages = append(errorMessages, err.Error())
		}
		return fmt.Errorf("deletion errors: %s", strings.Join(errorMessages, "; "))
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
	cmd := exec.Command(m.BinaryPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := "failed to create node '%s': %s"
		logger.Errorln(errMsg, name, stderr.String())
		return fmt.Errorf(errMsg, name, stderr.String())
	}

	return nil
}

func (m *MultipassClient) DeleteNode(name string) error {
	cmd := exec.Command(m.BinaryPath, "delete", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := fmt.Sprintf("Failed to delete node '%s': %s", name, stderr.String())
		logger.Errorln(errMsg)
		return fmt.Errorf("failed to delete node '%s': %s - %w", name, stderr.String(), err)
	}

	logger.Debugln("Successfully deleted node '%s'", name)
	return nil
}

func (m *MultipassClient) PurgeNodes() error {
	logger.Infoln("Purging deleted nodes")
	cmd := exec.Command(m.BinaryPath, "purge")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := fmt.Sprintf("Failed to purge deleted instances: %s", stderr.String())
		logger.Errorln(errMsg)
		return fmt.Errorf("failed to purge deleted instances: %s - %w", stderr.String(), err)
	}

	logger.Successln("Successfully purged deleted nodes")
	return nil
}

func (m *MultipassClient) GetNodeIP(name string) (string, error) {
	cmd := exec.Command(m.BinaryPath, "info", name, "--format", "json")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := fmt.Sprintf("Failed to get IP address for node '%s': %s", name, stderr.String())
		logger.Errorln(errMsg)
		return "", fmt.Errorf("failed to get IP address for node '%s': %s - %w", name, stderr.String(), err)
	}
	var data MultiPassInfo
	if err := json.Unmarshal(stdout.Bytes(), &data); err != nil {
		errMsg := fmt.Sprintf("Failed to parse JSON output: %s", err)
		logger.Errorln(errMsg)
		return "", fmt.Errorf("failed to parse JSON output: %s - %w", err, err)
	}
	ipv4List := data.Info.PlaygroundMaster.IPv4
	if len(ipv4List) == 0 {
		errMsg := fmt.Sprintf("No IPv4 addresses found for node '%s'", name)
		logger.Errorln(errMsg)
		return "", fmt.Errorf("no IPv4 addresses found for node '%s'", name)
	}
	ip := ipv4List[0]

	return ip, nil
}

func (m *MultipassClient) ExcuteShell(name string, command string) (string, error) {
	return m.ExcuteShellWithTimeout(name, command, 0) // No timeout by default
}

func (m *MultipassClient) ExcuteShellWithTimeout(name string, command string, timeoutSeconds int, envs ...string) (string, error) {
	ctx := context.Background()
	var cancel context.CancelFunc

	if timeoutSeconds > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, m.BinaryPath, "exec", name, "--", "bash", "-c", command)
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
