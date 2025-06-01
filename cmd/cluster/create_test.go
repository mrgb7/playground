package cluster

import (
	"testing"

	"github.com/mrgb7/playground/types"
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

func TestValidateCPUCount(t *testing.T) {
	tests := []struct {
		name        string
		cpus        int
		nodeType    string
		expectError bool
	}{
		{"valid master CPU", 2, "master", false},
		{"valid worker CPU", 1, "worker", false},
		{"maximum CPU", 32, "master", false},
		{"zero CPU", 0, "master", true},
		{"negative CPU", -1, "worker", true},
		{"excessive CPU", 33, "master", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := types.ValidateCPUCount(tt.cpus, tt.nodeType)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for %d CPUs but got none", tt.cpus)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for %d CPUs: %v", tt.cpus, err)
			}
		})
	}
}

func TestValidateMemoryFormat(t *testing.T) {
	tests := []struct {
		name        string
		memory      string
		nodeType    string
		expectError bool
	}{
		{"valid memory G", "2G", "master", false},
		{"valid memory M", "1024M", "worker", false},
		{"invalid format K", "2K", "master", true},
		{"invalid format no unit", "2", "master", true},
		{"invalid format lowercase", "2g", "master", true},
		{"invalid format empty", "", "master", true},
		{"invalid format text", "abc", "master", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := types.ValidateMemoryFormat(tt.memory, tt.nodeType)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for memory '%s' but got none", tt.memory)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for memory '%s': %v", tt.memory, err)
			}
		})
	}
}

func TestValidateDiskFormat(t *testing.T) {
	tests := []struct {
		name        string
		disk        string
		nodeType    string
		expectError bool
	}{
		{"valid disk G", "20G", "master", false},
		{"valid disk M", "1024M", "worker", false},
		{"valid disk T", "1T", "master", false},
		{"invalid format K", "20K", "master", true},
		{"invalid format no unit", "20", "master", true},
		{"invalid format lowercase", "20g", "master", true},
		{"invalid format empty", "", "master", true},
		{"invalid format text", "abc", "master", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := types.ValidateDiskFormat(tt.disk, tt.nodeType)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for disk '%s' but got none", tt.disk)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for disk '%s': %v", tt.disk, err)
			}
		})
	}
}
