/*
 *
 * Copyright © 2025 Dell Inc. or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *      http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package retriever

import (
	"context"
	"errors"
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

// Mock function to return a simulated successful clientset
func FakeGetClientset() (kubernetes.Interface, error) {
	return fake.NewSimpleClientset(), nil
}

// Mock function to return a simulated error when creating clientset
func FakeGetClientsetError() (kubernetes.Interface, error) {
	return nil, errors.New("simulated clientset creation error")
}

// Mock the InClusterConfig function
func mockInClusterConfig() (*rest.Config, error) {
	return &rest.Config{}, nil
}

// Mock the InClusterConfig function to return an error
func mockInClusterConfigError() (*rest.Config, error) {
	return nil, errors.New("mock error")
}

func createTestClient(fakeClientset func() (kubernetes.Interface, error)) *MetadataRetrieverClientType {
	client := NewMetadataRetrieverClient(nil, 0)
	client.getClientset = fakeClientset
	return client
}

func TestDefaultGetClientset(t *testing.T) {
	// Test the successful case
	restInClusterConfig = mockInClusterConfig
	defer func() {
		restInClusterConfig = rest.InClusterConfig
	}()

	clientset, err := defaultGetClientset()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if clientset == nil {
		t.Fatalf("expected clientset, got nil")
	}

	// Check if the clientset is of the correct type
	if _, ok := clientset.(*kubernetes.Clientset); !ok {
		t.Fatalf("expected *kubernetes.Clientset, got %T", clientset)
	}

	// Test the error case
	restInClusterConfig = mockInClusterConfigError

	clientset, err = defaultGetClientset()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if clientset != nil {
		t.Fatalf("expected nil clientset, got %v", clientset)
	}

	// Check if the error message is as expected
	expectedError := "mock error"
	if err.Error() != expectedError {
		t.Fatalf("expected error message %q, got %q", expectedError, err.Error())
	}
}

func TestGetPVCLabels_EmptyName(t *testing.T) {
	client := createTestClient(FakeGetClientset)
	req := &GetPVCLabelsRequest{Name: "", NameSpace: "default"}

	_, err := client.GetPVCLabels(context.Background(), req)
	if err == nil || err.Error() != "PVC Name cannot be empty" {
		t.Fatalf("expected error: PVC Name cannot be empty, got: %v", err)
	}
}

func TestGetPVCLabels_ErrorCreatingClientset(t *testing.T) {
	client := createTestClient(FakeGetClientsetError)
	req := &GetPVCLabelsRequest{Name: "mypvc", NameSpace: "default"}

	_, err := client.GetPVCLabels(context.Background(), req)
	if err == nil || err.Error() != "simulated clientset creation error" {
		t.Fatalf("expected error: simulated clientset creation error, got: %v", err)
	}
}

func TestGetPVCLabels_ErrorRetrievingPVCInfo(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset()

	client := NewMetadataRetrieverClient(nil, 0)
	client.getClientset = func() (kubernetes.Interface, error) {
		return fakeClientset, nil
	}
	req := &GetPVCLabelsRequest{Name: "nonexistent", NameSpace: "default"}

	_, err := client.GetPVCLabels(context.Background(), req)
	expectedErrorSnippet := "not found"
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected an error including \"%s\", but got \"%v\"", expectedErrorSnippet, err)
	}
}

func TestGetPVCLabels_Success(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset(&v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mypvc",
			Namespace: "default",
			Labels:    map[string]string{"key1": "value1", "key2": "value2"},
		},
	})

	client := NewMetadataRetrieverClient(nil, 0)
	client.getClientset = func() (kubernetes.Interface, error) {
		return fakeClientset, nil
	}
	req := &GetPVCLabelsRequest{Name: "mypvc", NameSpace: "default"}

	resp, err := client.GetPVCLabels(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(resp.Parameters) != 2 || resp.Parameters["key1"] != "value1" || resp.Parameters["key2"] != "value2" {
		t.Fatalf("expected map[key1:value1 key2:value2], got: %v", resp.Parameters)
	}
}

func TestNewMetadataRetrieverClient(t *testing.T) {
	// Test case: NewMetadataRetrieverClient
	client := NewMetadataRetrieverClient(nil, 0)
	if client == nil {
		t.Error("Expected client to be created")
	}
}
