package validator

import (
	"testing"
)

func TestGetAvailableCPU(t *testing.T) {
	cpu, err := GetAvailableCPU()
	if err != nil {
		t.Errorf("GetAvailableCPU() error = %v", err)
	}
	if cpu <= 0 {
		t.Errorf("GetAvailableCPU() = %v, want > 0", cpu)
	}
}

func TestGetAvailableMemory(t *testing.T) {
	mem, err := GetAvailableMemory()
	if err != nil {
		t.Errorf("GetAvailableMemory() error = %v", err)
	}
	if mem <= 0 {
		t.Errorf("GetAvailableMemory() = %v, want > 0", mem)
	}
}

func TestGetAvailableDisk(t *testing.T) {
	disk, err := GetAvailableDisk()
	if err != nil {
		t.Errorf("GetAvailableDisk() error = %v", err)
	}
	if disk < 0 {
		t.Errorf("GetAvailableDisk() = %v, want >= 0", disk)
	}
}

func TestIsPortInUse(t *testing.T) {
	// Test with a random high port that's likely to be free
	port := 54321
	if IsPortInUse(port) {
		t.Errorf("IsPortInUse(%d) = true, want false", port)
	}

	// Test with a well-known port that's likely to be in use
	if !IsPortInUse(22) { // SSH port
		t.Errorf("IsPortInUse(22) = false, want true")
	}
}
