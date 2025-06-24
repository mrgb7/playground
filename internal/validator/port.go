package validator

import (
	"fmt"
	"net"
)

const RequiredPort = 6443

type PortStatus struct {
	IsOpen bool
	Error  error
}

func CheckPortAvailability() (*PortStatus, error) {
	status := &PortStatus{}

	addr := fmt.Sprintf(":%d", RequiredPort)
	listener, err := net.Listen("tcp", addr)

	if err != nil {
		status.IsOpen = false
		status.Error = err
	} else {
		status.IsOpen = true
		listener.Close()
	}

	return status, nil
}

func ValidatePorts() (*ResourceStatus, error) {
	status, err := CheckPortAvailability()
	if err != nil {
		return nil, fmt.Errorf("failed to check port availability: %w", err)
	}

	result := &ResourceStatus{
		IsValid:         true,
		Messages:        []string{},
		Recommendations: []string{},
	}

	if !status.IsOpen {
		result.IsValid = false
		result.Messages = append(result.Messages,
			fmt.Sprintf("❌ Port %d is already in use", RequiredPort))
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("Free up port %d or configure a different port", RequiredPort))
	} else {
		result.Messages = append(result.Messages,
			fmt.Sprintf("✅ Port %d is available", RequiredPort))
	}

	return result, nil
}
