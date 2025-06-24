package validator

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type ResourceRequirements struct {
	MinCPU    int     // Minimum CPU cores required
	MinMemory float64 // Minimum memory in GB
	MinDisk   float64 // Minimum disk space in GB
}

type ResourceStatus struct {
	AvailableCPU    int      // Available CPU cores
	AvailableMemory float64  // Available memory in GB
	AvailableDisk   float64  // Available disk space in GB
	IsValid         bool     // Overall validation status
	Messages        []string // Validation messages
	Recommendations []string // Recommendations for fixing issues
}

func CalculateResourceRequirements(masterCPUs int, masterMemory, masterDisk string, workerCPUs int, workerMemory, workerDisk string, workerCount int) (*ResourceRequirements, error) {

	masterMemGB, err := parseMemoryToGB(masterMemory)
	if err != nil {
		return nil, fmt.Errorf("invalid master memory format: %w", err)
	}

	workerMemGB, err := parseMemoryToGB(workerMemory)
	if err != nil {
		return nil, fmt.Errorf("invalid worker memory format: %w", err)
	}

	// Parse disk values
	masterDiskGB, err := parseDiskToGB(masterDisk)
	if err != nil {
		return nil, fmt.Errorf("invalid master disk format: %w", err)
	}

	workerDiskGB, err := parseDiskToGB(workerDisk)
	if err != nil {
		return nil, fmt.Errorf("invalid worker disk format: %w", err)
	}

	totalCPU := masterCPUs + (workerCPUs * workerCount)
	totalMemory := masterMemGB + (workerMemGB * float64(workerCount))
	totalDisk := masterDiskGB + (workerDiskGB * float64(workerCount))

	return &ResourceRequirements{
		MinCPU:    totalCPU,
		MinMemory: totalMemory,
		MinDisk:   totalDisk,
	}, nil
}

func parseMemoryToGB(memory string) (float64, error) {
	matched, err := regexp.MatchString(`^[0-9]+[GM]$`, memory)
	if err != nil {
		return 0, fmt.Errorf("error validating memory format: %w", err)
	}
	if !matched {
		return 0, fmt.Errorf("memory must be in format like '2G' or '1024M'")
	}

	value, err := strconv.ParseFloat(memory[:len(memory)-1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid memory value: %w", err)
	}

	unit := strings.ToUpper(memory[len(memory)-1:])
	if unit == "M" {
		value = value / 1024 // Convert MB to GB
	}

	return value, nil
}

func parseDiskToGB(disk string) (float64, error) {
	matched, err := regexp.MatchString(`^[0-9]+[GMT]$`, disk)
	if err != nil {
		return 0, fmt.Errorf("error validating disk format: %w", err)
	}
	if !matched {
		return 0, fmt.Errorf("disk must be in format like '20G', '1024M', or '1T'")
	}

	value, err := strconv.ParseFloat(disk[:len(disk)-1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid disk value: %w", err)
	}

	unit := strings.ToUpper(disk[len(disk)-1:])
	switch unit {
	case "M":
		value = value / 1024 // Convert MB to GB
	case "T":
		value = value * 1024 // Convert TB to GB
	}

	return value, nil
}

func ValidateResources(requirements *ResourceRequirements) (*ResourceStatus, error) {
	status := &ResourceStatus{
		Messages:        make([]string, 0),
		Recommendations: make([]string, 0),
	}

	// Get current system resources
	cpu, err := GetAvailableCPU()
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU info: %w", err)
	}
	status.AvailableCPU = cpu

	memory, err := GetAvailableMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory info: %w", err)
	}
	status.AvailableMemory = memory

	disk, err := GetAvailableDisk()
	if err != nil {
		return nil, fmt.Errorf("failed to get disk info: %w", err)
	}
	status.AvailableDisk = disk

	// Set initial validation status
	status.IsValid = true

	if status.AvailableCPU < requirements.MinCPU {
		status.Messages = append(status.Messages,
			fmt.Sprintf("❌ CPU: %d cores available (%d required)",
				status.AvailableCPU, requirements.MinCPU))
		status.Recommendations = append(status.Recommendations,
			fmt.Sprintf("Ensure at least %d CPU cores are available", requirements.MinCPU))
		status.IsValid = false
	} else {
		status.Messages = append(status.Messages,
			fmt.Sprintf("✅ CPU: %d cores available (%d required)",
				status.AvailableCPU, requirements.MinCPU))
	}

	if status.AvailableMemory < requirements.MinMemory {
		status.Messages = append(status.Messages,
			fmt.Sprintf("❌ Memory: %.1f GB available (%.1f GB required)",
				status.AvailableMemory, requirements.MinMemory))
		status.Recommendations = append(status.Recommendations,
			fmt.Sprintf("Free up at least %.1f GB of memory to meet minimum requirements",
				requirements.MinMemory-status.AvailableMemory))
		status.Recommendations = append(status.Recommendations,
			"Close unnecessary applications to free memory")
		status.IsValid = false
	} else {
		status.Messages = append(status.Messages,
			fmt.Sprintf("✅ Memory: %.1f GB available (%.1f GB required)",
				status.AvailableMemory, requirements.MinMemory))
	}

	if status.AvailableDisk < requirements.MinDisk {
		status.Messages = append(status.Messages,
			fmt.Sprintf("❌ Disk: %.1f GB available (%.1f GB required)",
				status.AvailableDisk, requirements.MinDisk))
		status.Recommendations = append(status.Recommendations,
			fmt.Sprintf("Free up at least %.1f GB of disk space to meet minimum requirements",
				requirements.MinDisk-status.AvailableDisk))
		status.Recommendations = append(status.Recommendations,
			"Consider cleaning up disk space for optimal performance")
		status.IsValid = false
	} else {
		status.Messages = append(status.Messages,
			fmt.Sprintf("✅ Disk: %.1f GB available (%.1f GB required)",
				status.AvailableDisk, requirements.MinDisk))
	}

	return status, nil
}
