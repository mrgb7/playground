package validator

import (
	"strings"
	"testing"
)

// Save original functions
var (
	originalGetCPU      = getAvailableCPU
	originalGetMemory   = getAvailableMemory
	originalGetDisk     = getAvailableDisk
	originalIsPortInUse = isPortInUse
)

func TestResourceValidationIntegration(t *testing.T) {
	// Restore original functions after test
	defer func() {
		getAvailableCPU = originalGetCPU
		getAvailableMemory = originalGetMemory
		getAvailableDisk = originalGetDisk
		isPortInUse = originalIsPortInUse
	}()

	tests := []struct {
		name           string
		masterCPU      int
		masterMemory   string
		masterDisk     string
		workerCPU      int
		workerMemory   string
		workerDisk     string
		workerCount    int
		mockCPU        int
		mockMemory     float64
		mockDisk       float64
		mockPortInUse  bool
		expectValid    bool
		expectErrors   int
		expectWarnings int
	}{
		{
			name:           "Single node cluster with sufficient resources",
			masterCPU:      2,
			masterMemory:   "4G",
			masterDisk:     "20G",
			workerCPU:      0,
			workerMemory:   "0G",
			workerDisk:     "0G",
			workerCount:    0,
			mockCPU:        4,
			mockMemory:     8.0,
			mockDisk:       40.0,
			mockPortInUse:  false,
			expectValid:    true,
			expectErrors:   0,
			expectWarnings: 0,
		},
		{
			name:           "Multi-node cluster with insufficient CPU",
			masterCPU:      2,
			masterMemory:   "4G",
			masterDisk:     "20G",
			workerCPU:      2,
			workerMemory:   "4G",
			workerDisk:     "20G",
			workerCount:    2,
			mockCPU:        4,
			mockMemory:     16.0,
			mockDisk:       80.0,
			mockPortInUse:  false,
			expectValid:    false,
			expectErrors:   1,
			expectWarnings: 0,
		},
		{
			name:           "Multi-node cluster with insufficient memory",
			masterCPU:      2,
			masterMemory:   "4G",
			masterDisk:     "20G",
			workerCPU:      2,
			workerMemory:   "4G",
			workerDisk:     "20G",
			workerCount:    2,
			mockCPU:        8,
			mockMemory:     8.0,
			mockDisk:       80.0,
			mockPortInUse:  false,
			expectValid:    false,
			expectErrors:   1,
			expectWarnings: 0,
		},
		{
			name:           "Large multi-node cluster with insufficient disk",
			masterCPU:      4,
			masterMemory:   "8G",
			masterDisk:     "40G",
			workerCPU:      4,
			workerMemory:   "8G",
			workerDisk:     "40G",
			workerCount:    3,
			mockCPU:        16,
			mockMemory:     32.0,
			mockDisk:       100.0,
			mockPortInUse:  true,
			expectValid:    false,
			expectErrors:   2,
			expectWarnings: 0,
		},
		{
			name:           "Large multi-node cluster with port in use",
			masterCPU:      4,
			masterMemory:   "8G",
			masterDisk:     "40G",
			workerCPU:      4,
			workerMemory:   "8G",
			workerDisk:     "40G",
			workerCount:    3,
			mockCPU:        16,
			mockMemory:     32.0,
			mockDisk:       120.0,
			mockPortInUse:  true,
			expectValid:    false,
			expectErrors:   2,
			expectWarnings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up mocks
			getAvailableCPU = func() (int, error) { return tt.mockCPU, nil }
			getAvailableMemory = func() (float64, error) { return tt.mockMemory, nil }
			getAvailableDisk = func() (float64, error) { return tt.mockDisk, nil }
			isPortInUse = func(port int) bool { return tt.mockPortInUse }

			// Calculate resource requirements
			requirements, err := CalculateResourceRequirements(
				tt.masterCPU, tt.masterMemory, tt.masterDisk,
				tt.workerCPU, tt.workerMemory, tt.workerDisk,
				tt.workerCount,
			)
			if err != nil {
				t.Errorf("CalculateResourceRequirements() error = %v", err)
				return
			}

			// Validate resources
			status, err := ValidateResources(requirements)
			if err != nil {
				t.Errorf("ValidateResources() error = %v", err)
				return
			}

			// Validate ports
			portStatus, err := ValidatePorts()
			if err != nil {
				t.Errorf("ValidatePorts() error = %v", err)
				return
			}

			// Check overall validation status
			isValid := status.IsValid && portStatus.IsValid
			if isValid != tt.expectValid {
				t.Errorf("Validation status = %v, want %v", isValid, tt.expectValid)
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
			for _, msg := range portStatus.Messages {
				if strings.HasPrefix(msg, "❌") {
					errorCount++
				} else if strings.HasPrefix(msg, "⚠️") {
					warningCount++
				}
			}

			if errorCount != tt.expectErrors {
				t.Errorf("Got %d errors, want %d", errorCount, tt.expectErrors)
			}

			if warningCount != tt.expectWarnings {
				t.Errorf("Got %d warnings, want %d", warningCount, tt.expectWarnings)
			}
		})
	}
}
