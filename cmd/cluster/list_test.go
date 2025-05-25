package cluster

import (
	"testing"

	"github.com/mrgb7/playground/internal/multipass"
)

func TestListClusters(t *testing.T) {
	client := multipass.NewMultipassClient()
	
	// Check if multipass is installed before running the test
	if !client.IsMultipassInstalled() {
		t.Skip("Multipass is not installed, skipping test")
	}

	clusters, err := client.ListClusters()
	if err != nil {
		t.Fatalf("ListClusters failed: %v", err)
	}

	// The test should pass regardless of whether clusters exist or not
	// We just verify that the method doesn't return an error
	t.Logf("Found %d clusters: %v", len(clusters), clusters)
} 