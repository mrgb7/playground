package cluster

import (
	"testing"
)

func TestConstants(t *testing.T) {
	// Test that command constants are properly defined
	if K3sCreateMasterCmd == "" {
		t.Error("K3sCreateMasterCmd should not be empty")
	}
	
	if GetAccessTokenCmd == "" {
		t.Error("GetAccessTokenCmd should not be empty")
	}
	
	if K3sCreateWorkerCmd == "" {
		t.Error("K3sCreateWorkerCmd should not be empty")
	}
	
	if KubeConfigCmd == "" {
		t.Error("KubeConfigCmd should not be empty")
	}
	
	if K3sInstallTimeout <= 0 {
		t.Errorf("K3sInstallTimeout should be positive, got: %d", K3sInstallTimeout)
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