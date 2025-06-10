package validator

import (
	"fmt"
	"net"
	"os"
	"runtime"

	"github.com/shirou/gopsutil/v3/mem"
)

// Platform functions that can be mocked in tests
var (
	GetAvailableCPU    = getAvailableCPU
	GetAvailableMemory = getAvailableMemory
	GetAvailableDisk   = getAvailableDisk
	IsPortInUse        = isPortInUse
)

func getAvailableCPU() (int, error) {
	return runtime.NumCPU(), nil
}

func getAvailableMemory() (float64, error) {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return 0, fmt.Errorf("failed to get system memory info: %w", err)
	}

	return float64(vmStat.Total) / (1024 * 1024 * 1024), nil
}

func getAvailableDisk() (float64, error) {
	wd, err := os.Getwd()
	if err != nil {
		return 0, fmt.Errorf("failed to get working directory: %w", err)
	}

	info, err := os.Stat(wd)
	if err != nil {
		return 0, fmt.Errorf("failed to get filesystem stats: %w", err)
	}

	available := float64(info.Size())
	return available / 1024 / 1024 / 1024, nil
}

func isPortInUse(port int) bool {
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return true
	}
	if err := listener.Close(); err != nil {
		fmt.Printf("Warning: failed to close listener: %v\n", err)
	}
	return false
}
