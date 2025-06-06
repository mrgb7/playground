package validator

import (
	"testing"
)

func TestGetAvailableCPU(t *testing.T) {
	cpu, err := getAvailableCPU()
	if err != nil {
		t.Errorf("getAvailableCPU() error = %v", err)
	}
	if cpu <= 0 {
		t.Errorf("getAvailableCPU() = %v, want > 0", cpu)
	}
}

func TestGetAvailableMemory(t *testing.T) {
	memory, err := getAvailableMemory()
	if err != nil {
		t.Errorf("getAvailableMemory() error = %v", err)
	}
	if memory <= 0 {
		t.Errorf("getAvailableMemory() = %v, want > 0", memory)
	}
}

func TestGetAvailableDisk(t *testing.T) {
	disk, err := getAvailableDisk()
	if err != nil {
		t.Errorf("getAvailableDisk() error = %v", err)
	}
	if disk <= 0 {
		t.Errorf("getAvailableDisk() = %v, want > 0", disk)
	}
}

func TestIsPortInUse(t *testing.T) {
	testPort := 65432
	if isPortInUse(testPort) {
		t.Errorf("isPortInUse(%d) = true, want false for unused port", testPort)
	}

	if isPortInUse(0) {
		t.Error("isPortInUse(0) = true, want false for port 0")
	}

	_ = isPortInUse(6443)
}
