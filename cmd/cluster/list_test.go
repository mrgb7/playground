package cluster

import (
	"testing"

	"github.com/mrgb7/playground/internal/multipass"
)

func TestListClusters(t *testing.T) {
	client := multipass.NewMultipassClient()
	
	if !client.IsMultipassInstalled() {
		t.Skip("Multipass is not installed, skipping test")
	}

	clusters, err := client.ListClusters()
	if err != nil {
		t.Fatalf("ListClusters failed: %v", err)
	}

	t.Logf("Found %d clusters: %v", len(clusters), clusters)
} 