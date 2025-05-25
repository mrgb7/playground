package plugins

import (
	"testing"
)

func TestInstallerTrackerConstants(t *testing.T) {
	if InstallerTrackerConfigMapName == "" {
		t.Error("InstallerTrackerConfigMapName should not be empty")
	}

	if InstallerTrackerNamespace == "" {
		t.Error("InstallerTrackerNamespace should not be empty")
	}

	if InstallerTypeHelm == "" {
		t.Error("InstallerTypeHelm should not be empty")
	}

	if InstallerTypeArgoCD == "" {
		t.Error("InstallerTypeArgoCD should not be empty")
	}

	// Verify expected values
	if InstallerTrackerConfigMapName != "playground-plugin-installer-tracker" {
		t.Errorf("Expected InstallerTrackerConfigMapName to be 'playground-plugin-installer-tracker', got '%s'", InstallerTrackerConfigMapName)
	}

	if InstallerTrackerNamespace != "kube-system" {
		t.Errorf("Expected InstallerTrackerNamespace to be 'kube-system', got '%s'", InstallerTrackerNamespace)
	}

	if InstallerTypeHelm != "helm" {
		t.Errorf("Expected InstallerTypeHelm to be 'helm', got '%s'", InstallerTypeHelm)
	}

	if InstallerTypeArgoCD != "argocd" {
		t.Errorf("Expected InstallerTypeArgoCD to be 'argocd', got '%s'", InstallerTypeArgoCD)
	}
}

func TestNewInstallerTracker(t *testing.T) {
	tests := []struct {
		name        string
		kubeConfig  string
		expectError bool
	}{
		{
			name:        "invalid kubeconfig",
			kubeConfig:  "invalid-config",
			expectError: true,
		},
		{
			name:        "empty kubeconfig",
			kubeConfig:  "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker, err := NewInstallerTracker(tt.kubeConfig)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError && tracker == nil {
				t.Errorf("Expected tracker but got nil")
			}

			if tt.expectError && tracker != nil {
				t.Errorf("Expected nil tracker but got one")
			}
		})
	}
}

func TestInstallerTrackerStructure(t *testing.T) {
	// Test that we can create the tracker struct (even with invalid config for structure test)
	tracker := &InstallerTracker{
		kubeConfig: "test-config",
		k8sClient:  nil, // Will be nil with invalid config, which is fine for structure test
	}

	if tracker.kubeConfig != "test-config" {
		t.Errorf("Expected kubeConfig to be 'test-config', got '%s'", tracker.kubeConfig)
	}
}
