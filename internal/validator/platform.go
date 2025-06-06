package validator

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"syscall"
)

// Platform functions that can be mocked in tests
var (
	getAvailableCPU    = getAvailableCPUImpl
	getAvailableMemory = getAvailableMemoryImpl
	getAvailableDisk   = getAvailableDiskImpl
	isPortInUse        = isPortInUseImpl
)

func getAvailableCPUImpl() (int, error) {
	return runtime.NumCPU(), nil
}

func getAvailableMemoryImpl() (float64, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	var si syscall.Sysinfo_t
	if err := syscall.Sysinfo(&si); err != nil {
		return 0, fmt.Errorf("failed to get system info: %w", err)
	}

	totalMemory := float64(si.Totalram) / (1024 * 1024 * 1024)
	return totalMemory, nil
}

func getAvailableDiskImpl() (float64, error) {
	var stat syscall.Statfs_t
	wd, err := os.Getwd()
	if err != nil {
		return 0, fmt.Errorf("failed to get working directory: %w", err)
	}

	if err := syscall.Statfs(wd, &stat); err != nil {
		return 0, fmt.Errorf("failed to get filesystem stats: %w", err)
	}

	available := stat.Bavail * uint64(stat.Bsize)
	availableGB := float64(available) / (1024 * 1024 * 1024)
	return availableGB, nil
}

func isPortInUseImpl(port int) bool {
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return true
	}

	listener.Close()
	return false
}
