package validator

import (
	"strings"
	"testing"
)

func TestCalculateResourceRequirements(t *testing.T) {
	tests := []struct {
		name         string
		masterCPUs   int
		masterMemory string
		masterDisk   string
		workerCPUs   int
		workerMemory string
		workerDisk   string
		workerCount  int
		expectCPU    int
		expectMemory float64
		expectDisk   float64
		expectError  bool
	}{
		{
			name:         "Single node cluster",
			masterCPUs:   2,
			masterMemory: "2G",
			masterDisk:   "20G",
			workerCPUs:   2,
			workerMemory: "2G",
			workerDisk:   "20G",
			workerCount:  0,
			expectCPU:    2,
			expectMemory: 2,
			expectDisk:   20,
			expectError:  false,
		},
		{
			name:         "Multi-node cluster",
			masterCPUs:   4,
			masterMemory: "4G",
			masterDisk:   "40G",
			workerCPUs:   2,
			workerMemory: "2G",
			workerDisk:   "20G",
			workerCount:  2,
			expectCPU:    8,
			expectMemory: 8,
			expectDisk:   80,
			expectError:  false,
		},
		{
			name:         "Invalid memory format",
			masterCPUs:   2,
			masterMemory: "2K",
			masterDisk:   "20G",
			workerCPUs:   2,
			workerMemory: "2G",
			workerDisk:   "20G",
			workerCount:  0,
			expectError:  true,
		},
		{
			name:         "Invalid disk format",
			masterCPUs:   2,
			masterMemory: "2G",
			masterDisk:   "20K",
			workerCPUs:   2,
			workerMemory: "2G",
			workerDisk:   "20G",
			workerCount:  0,
			expectError:  true,
		},
		{
			name:         "Memory in MB",
			masterCPUs:   2,
			masterMemory: "2048M",
			masterDisk:   "20G",
			workerCPUs:   2,
			workerMemory: "2G",
			workerDisk:   "20G",
			workerCount:  0,
			expectCPU:    2,
			expectMemory: 2,
			expectDisk:   20,
			expectError:  false,
		},
		{
			name:         "Disk in TB",
			masterCPUs:   2,
			masterMemory: "2G",
			masterDisk:   "1T",
			workerCPUs:   2,
			workerMemory: "2G",
			workerDisk:   "20G",
			workerCount:  0,
			expectCPU:    2,
			expectMemory: 2,
			expectDisk:   1024,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requirements, err := CalculateResourceRequirements(
				tt.masterCPUs, tt.masterMemory, tt.masterDisk,
				tt.workerCPUs, tt.workerMemory, tt.workerDisk,
				tt.workerCount,
			)

			if tt.expectError {
				if err == nil {
					t.Errorf("CalculateResourceRequirements() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("CalculateResourceRequirements() error = %v", err)
				return
			}

			if requirements.MinCPU != tt.expectCPU {
				t.Errorf("MinCPU = %v, want %v", requirements.MinCPU, tt.expectCPU)
			}

			if requirements.MinMemory != tt.expectMemory {
				t.Errorf("MinMemory = %v, want %v", requirements.MinMemory, tt.expectMemory)
			}

			if requirements.MinDisk != tt.expectDisk {
				t.Errorf("MinDisk = %v, want %v", requirements.MinDisk, tt.expectDisk)
			}
		})
	}
}

func TestValidateResources(t *testing.T) {
	// Mock the platform-specific functions for testing
	originalGetCPU := GetAvailableCPU
	originalGetMemory := GetAvailableMemory
	originalGetDisk := GetAvailableDisk
	defer func() {
		GetAvailableCPU = originalGetCPU
		GetAvailableMemory = originalGetMemory
		GetAvailableDisk = originalGetDisk
	}()

	tests := []struct {
		name           string
		requirements   *ResourceRequirements
		mockCPU        int
		mockMemory     float64
		mockDisk       float64
		expectValid    bool
		expectErrors   int
		expectWarnings int
	}{
		{
			name: "Sufficient resources",
			requirements: &ResourceRequirements{
				MinCPU:    2,
				MinMemory: 4,
				MinDisk:   10,
			},
			mockCPU:        4,
			mockMemory:     8,
			mockDisk:       20,
			expectValid:    true,
			expectErrors:   0,
			expectWarnings: 0,
		},
		{
			name: "Insufficient CPU",
			requirements: &ResourceRequirements{
				MinCPU:    4,
				MinMemory: 4,
				MinDisk:   10,
			},
			mockCPU:        2,
			mockMemory:     8,
			mockDisk:       20,
			expectValid:    false,
			expectErrors:   1,
			expectWarnings: 0,
		},
		{
			name: "Insufficient memory",
			requirements: &ResourceRequirements{
				MinCPU:    2,
				MinMemory: 8,
				MinDisk:   10,
			},
			mockCPU:        4,
			mockMemory:     4,
			mockDisk:       20,
			expectValid:    false,
			expectErrors:   1,
			expectWarnings: 0,
		},
		{
			name: "Insufficient disk",
			requirements: &ResourceRequirements{
				MinCPU:    2,
				MinMemory: 4,
				MinDisk:   20,
			},
			mockCPU:        4,
			mockMemory:     8,
			mockDisk:       10,
			expectValid:    false,
			expectErrors:   1,
			expectWarnings: 0,
		},
		{
			name: "Multiple insufficient resources",
			requirements: &ResourceRequirements{
				MinCPU:    4,
				MinMemory: 8,
				MinDisk:   20,
			},
			mockCPU:        2,
			mockMemory:     4,
			mockDisk:       10,
			expectValid:    false,
			expectErrors:   3,
			expectWarnings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up mocks
			GetAvailableCPU = func() (int, error) { return tt.mockCPU, nil }
			GetAvailableMemory = func() (float64, error) { return tt.mockMemory, nil }
			GetAvailableDisk = func() (float64, error) { return tt.mockDisk, nil }

			status, err := ValidateResources(tt.requirements)
			if err != nil {
				t.Errorf("ValidateResources() error = %v", err)
				return
			}

			if status.IsValid != tt.expectValid {
				t.Errorf("ValidateResources() IsValid = %v, want %v", status.IsValid, tt.expectValid)
			}

			// Count error and warning messages
			errorCount := 0
			warningCount := 0
			for _, msg := range status.Messages {
				if strings.HasPrefix(msg, "❌") {
					errorCount++
				} else if strings.HasPrefix(msg, "⚠️") {
					warningCount++
				}
			}

			if errorCount != tt.expectErrors {
				t.Errorf("ValidateResources() got %d errors, want %d", errorCount, tt.expectErrors)
			}

			if warningCount != tt.expectWarnings {
				t.Errorf("ValidateResources() got %d warnings, want %d", warningCount, tt.expectWarnings)
			}
		})
	}
}
