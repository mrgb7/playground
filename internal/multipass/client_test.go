package multipass

import (
	"testing"
)

func TestNewMultipassClient(t *testing.T) {
	client := NewMultipassClient()

	if client == nil {
		t.Fatal("NewMultipassClient() returned nil")
	}

	if client.BinaryPath != "multipass" {
		t.Errorf("Expected BinaryPath to be 'multipass', got: %s", client.BinaryPath)
	}
}

func TestMultipassClient_IsMultipassInstalled(t *testing.T) {
	client := NewMultipassClient()

	client.BinaryPath = "nonexistent-binary"
	if client.IsMultipassInstalled() {
		t.Error("Expected IsMultipassInstalled() to return false for nonexistent binary")
	}

	client.BinaryPath = "echo"
	if !client.IsMultipassInstalled() {
		t.Error("Expected IsMultipassInstalled() to return true for 'echo' command")
	}
}

func TestConstants(t *testing.T) {
	if DefaultMasterCPUs <= 0 {
		t.Errorf("DefaultMasterCPUs should be positive, got: %d", DefaultMasterCPUs)
	}

	if DefaultMasterMemory == "" {
		t.Error("DefaultMasterMemory should not be empty")
	}

	if DefaultMasterDisk == "" {
		t.Error("DefaultMasterDisk should not be empty")
	}

	if DefaultWorkerCPUs <= 0 {
		t.Errorf("DefaultWorkerCPUs should be positive, got: %d", DefaultWorkerCPUs)
	}

	if DefaultWorkerMemory == "" {
		t.Error("DefaultWorkerMemory should not be empty")
	}

	if DefaultWorkerDisk == "" {
		t.Error("DefaultWorkerDisk should not be empty")
	}
}

func TestMultipassClient_CreateNode_ValidatesInput(t *testing.T) {
	client := NewMultipassClient()
	client.BinaryPath = "nonexistent-binary" // Ensure it fails for the right reason

	err := client.CreateNode("", 1, "1G", "5G")
	if err == nil {
		t.Error("Expected CreateNode to fail with empty node name")
	}
}
