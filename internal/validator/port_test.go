package validator

import (
	"fmt"
	"net"
	"testing"
)

func TestCheckPortAvailability(t *testing.T) {
	status, err := CheckPortAvailability()
	if err != nil {
		t.Errorf("CheckPortAvailability() error = %v", err)
		return
	}

	if !status.IsOpen {
		t.Errorf("Port %d should be available", RequiredPort)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", RequiredPort))
	if err != nil {
		t.Fatalf("Failed to create test listener: %v", err)
	}
	defer listener.Close()

	status, err = CheckPortAvailability()
	if err != nil {
		t.Errorf("CheckPortAvailability() error = %v", err)
		return
	}

	if status.IsOpen {
		t.Error("Expected port to be in use")
	}
}

func TestValidatePorts(t *testing.T) {
	status, err := ValidatePorts()
	if err != nil {
		t.Errorf("ValidatePorts() error = %v", err)
		return
	}

	if !status.IsValid {
		t.Error("ValidatePorts() expected valid status")
	}

	foundSuccess := false
	for _, msg := range status.Messages {
		if msg == fmt.Sprintf("✅ Port %d is available", RequiredPort) {
			foundSuccess = true
			break
		}
	}
	if !foundSuccess {
		t.Error("ValidatePorts() missing success message")
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", RequiredPort))
	if err != nil {
		t.Fatalf("Failed to create test listener: %v", err)
	}
	defer listener.Close()

	status, err = ValidatePorts()
	if err != nil {
		t.Errorf("ValidatePorts() error = %v", err)
		return
	}

	if status.IsValid {
		t.Error("ValidatePorts() expected invalid status when port is in use")
	}

	foundError := false
	for _, msg := range status.Messages {
		if msg == fmt.Sprintf("❌ Port %d is already in use", RequiredPort) {
			foundError = true
			break
		}
	}
	if !foundError {
		t.Error("ValidatePorts() missing error message for port in use")
	}

	foundRecommendation := false
	for _, rec := range status.Recommendations {
		if rec == fmt.Sprintf("Free up port %d or configure a different port", RequiredPort) {
			foundRecommendation = true
			break
		}
	}
	if !foundRecommendation {
		t.Error("ValidatePorts() missing recommendation for port in use")
	}
}
