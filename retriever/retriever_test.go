package retriever

import (
	"testing"

	"golang.org/x/net/context"
)

func TestGetPVCLabels(t *testing.T) {
	// Create a new client
	client := NewMetadataRetrieverClient(nil, 0)
	// Test case: PVC Name is empty
	_, err := client.GetPVCLabels(context.Background(), &GetPVCLabelsRequest{})
	if err == nil {
		t.Error("Expected error for empty PVC Name")
	}

	_, err = client.GetPVCLabels(context.Background(), &GetPVCLabelsRequest{Name: "pvc"})
	if err == nil {
		t.Error("Expected error for empty PVC Name")
	}
}

func TestNewMetadataRetrieverClient(t *testing.T) {
	// Test case: NewMetadataRetrieverClient
	client := NewMetadataRetrieverClient(nil, 0)
	if client == nil {
		t.Error("Expected client to be created")
	}
}
