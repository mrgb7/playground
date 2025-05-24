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
	
	// Test with invalid binary path
	client.BinaryPath = "nonexistent-binary"
	if client.IsMultipassInstalled() {
		t.Error("Expected IsMultipassInstalled() to return false for nonexistent binary")
	}
	
	// Test with valid command (assuming 'echo' exists on most systems)
	client.BinaryPath = "echo"
	if !client.IsMultipassInstalled() {
		t.Error("Expected IsMultipassInstalled() to return true for 'echo' command")
	}
}

func TestFormatNodeName(t *testing.T) {
	tests := []struct {
		clusterName string
		nodeType    string
		index       int
		expected    string
	}{
		{"test-cluster", "master", 0, "test-cluster-master"},
		{"test-cluster", "worker", 1, "test-cluster-worker-1"},
		{"my-cluster", "worker", 2, "my-cluster-worker-2"},
	}
	
	for _, tt := range tests {
		var result string
		if tt.nodeType == "master" {
			result = tt.clusterName + "-master"
		} else {
			result = tt.clusterName + "-worker-" + string(rune(tt.index+'0'))
		}
		
		if result != tt.expected {
			t.Errorf("Expected node name %s, got: %s", tt.expected, result)
		}
	}
} 