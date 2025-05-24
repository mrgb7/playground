package cluster

import (
	"testing"
)

func TestConstants(t *testing.T) {
	// Test that command constants are properly defined
	if K3S_CREATE_MASTER_CMD == "" {
		t.Error("K3S_CREATE_MASTER_CMD should not be empty")
	}
	
	if GET_ACCESS_TOKEN_CMD == "" {
		t.Error("GET_ACCESS_TOKEN_CMD should not be empty")
	}
	
	if K3S_CREATE_WORKER_CMD == "" {
		t.Error("K3S_CREATE_WORKER_CMD should not be empty")
	}
	
	if KUBE_CONFIG_CMD == "" {
		t.Error("KUBE_CONFIG_CMD should not be empty")
	}
	
	if K3S_INSTALL_TIMEOUT <= 0 {
		t.Errorf("K3S_INSTALL_TIMEOUT should be positive, got: %d", K3S_INSTALL_TIMEOUT)
	}
}

func TestCreateCommandExists(t *testing.T) {
	if createCmd == nil {
		t.Fatal("createCmd should not be nil")
	}
	
	if createCmd.Use != "create" {
		t.Errorf("Expected createCmd.Use to be 'create', got: %s", createCmd.Use)
	}
	
	if createCmd.Short == "" {
		t.Error("createCmd.Short should not be empty")
	}
} 