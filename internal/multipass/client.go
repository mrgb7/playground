package multipass

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/mohamedragab2024/playground/pkg/logger"
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

func (m *MultipassClient) CreateCluster(clusterName string, nodeCount int) error {
	masterName := fmt.Sprintf("%s-master", clusterName)
	logger.Debugln("Creating master node: %s", masterName)
	err := m.CreateNode(masterName, 2, "2G", "10G")
	if err != nil {
		errMsg := "failed to create master node: %v"
		logger.Error(errMsg, err)
		return fmt.Errorf(errMsg, err)
	}

	for i := 1; i < nodeCount; i++ {
		nodeName := fmt.Sprintf("%s-worker-%d", clusterName, i)
		logger.Infoln("Creating worker node: %s", nodeName)
		err := m.CreateNode(nodeName, 1, "1G", "5G")
		if err != nil {
			logger.Errorln("failed to create worker node %d: %v", i, err)
			return fmt.Errorf("failed to create worker node %d: %v", i, err)
		}
	}

	logger.Debugln("Cluster %s created successfully", clusterName)
	return nil
}

func (m *MultipassClient) DeleteCluster(clusterName string) error {
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
	for _, instance := range list.List {
		if strings.HasPrefix(instance.Name, clusterName) {
			m.DeleteNode(instance.Name)
		}
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
	logger.Infoln("Deleting node: %s", name)
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

func (m *MultipassClient) ExcuteShellWithTimeout(name string, command string, timeoutSeconds int) (string, error) {
	logger.Infoln("Executing shell command on node: %s", name)

	ctx := context.Background()
	var cancel context.CancelFunc

	if timeoutSeconds > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, m.BinaryPath, "exec", name, "--", "bash", "-c", command)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return stdout.String(), fmt.Errorf("command timed out after %d seconds", timeoutSeconds)
		}
		errMsg := fmt.Sprintf("Failed to execute shell command on node '%s': %s", name, stderr.String())
		logger.Errorln(errMsg)
		return "", fmt.Errorf("failed to execute shell command on node '%s': %s - %w", name, stderr.String(), err)
	}

	return stdout.String(), nil
}
