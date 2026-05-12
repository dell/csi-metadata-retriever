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

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

type pvcNilClientset struct {
	*fake.Clientset
}

func (c *pvcNilClientset) CoreV1() corev1client.CoreV1Interface {
	return &pvcNilCoreV1{CoreV1Interface: c.Clientset.CoreV1()}
}

type pvcNilCoreV1 struct {
	corev1client.CoreV1Interface
}

func (c *pvcNilCoreV1) PersistentVolumeClaims(_ string) corev1client.PersistentVolumeClaimInterface {
	return nil
}

type pvListErrorClientset struct {
	*fake.Clientset
}

func (c *pvListErrorClientset) CoreV1() corev1client.CoreV1Interface {
	return &pvListErrorCoreV1{CoreV1Interface: c.Clientset.CoreV1()}
}

type pvListErrorCoreV1 struct {
	corev1client.CoreV1Interface
}

func (c *pvListErrorCoreV1) PersistentVolumes() corev1client.PersistentVolumeInterface {
	return &pvListErrorPVInterface{PersistentVolumeInterface: c.CoreV1Interface.PersistentVolumes()}
}

type pvListErrorPVInterface struct {
	corev1client.PersistentVolumeInterface
}

func (c *pvListErrorPVInterface) List(_ context.Context, _ metav1.ListOptions) (*v1.PersistentVolumeList, error) {
	return nil, errors.New("simulated PV list error")
}

type pvcGetErrorClientset struct {
	*fake.Clientset
}

func (c *pvcGetErrorClientset) CoreV1() corev1client.CoreV1Interface {
	return &pvcGetErrorCoreV1{CoreV1Interface: c.Clientset.CoreV1()}
}

type pvcGetErrorCoreV1 struct {
	corev1client.CoreV1Interface
}

func (c *pvcGetErrorCoreV1) PersistentVolumeClaims(namespace string) corev1client.PersistentVolumeClaimInterface {
	return &pvcGetErrorPVCInterface{PersistentVolumeClaimInterface: c.CoreV1Interface.PersistentVolumeClaims(namespace)}
}

type pvcGetErrorPVCInterface struct {
	corev1client.PersistentVolumeClaimInterface
}

func (c *pvcGetErrorPVCInterface) Get(_ context.Context, _ string, _ metav1.GetOptions) (*v1.PersistentVolumeClaim, error) {
	return nil, errors.New("simulated PVC get error")
}

// Mock function to return a simulated successful clientset
func FakeGetClientset() (kubernetes.Interface, error) {
	return fake.NewSimpleClientset(), nil
}

// Mock function to return a simulated error when creating clientset
func FakeGetClientsetError() (kubernetes.Interface, error) {
	return nil, errors.New("simulated clientset creation error")
}

// Mock function to return a clientset that fails on PV List
func FakeGetClientsetPVListError() (kubernetes.Interface, error) {
	return &pvListErrorClientset{Clientset: fake.NewSimpleClientset()}, nil
}

// Mock function to return a clientset that fails on PVC Get
func FakeGetClientsetPVCGetError() (kubernetes.Interface, error) {
	return &pvcGetErrorClientset{Clientset: fake.NewSimpleClientset()}, nil
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

func TestGetPVCLabels_PVCClientIsNil_CoversBranch(t *testing.T) {
	wrapped := &pvcNilClientset{Clientset: fake.NewSimpleClientset()}
	client := NewMetadataRetrieverClient(nil, 0)
	client.getClientset = func() (kubernetes.Interface, error) {
		return wrapped, nil
	}
	req := &GetPVCLabelsRequest{
		Name:      "mypvc",
		NameSpace: "default",
	}
	resp, err := client.GetPVCLabels(context.Background(), req)

	if resp != nil {
		t.Fatalf("expected resp to be nil when pvcClient is nil; got: %#v", resp)
	}
	if err != nil {
		t.Fatalf("expected err to be nil with current implementation; got: %v", err)
	}
}

func TestGetPVCLabelsByPVName_EmptyPVName(t *testing.T) {
	client := createTestClient(FakeGetClientset)
	req := &GetPVCLabelsByPVNameRequest{PVName: "", VolumeHandle: ""}

	resp, err := client.GetPVCLabelsByPVName(context.Background(), req)
	assert.Error(t, err)
	assert.Equal(t, "VolumeHandle is empty", err.Error())
	assert.Nil(t, resp)
}

func TestGetPVCLabelsByPVName_ErrorCreatingClientset(t *testing.T) {
	client := createTestClient(FakeGetClientsetError)
	req := &GetPVCLabelsByPVNameRequest{PVName: "test-pv", VolumeHandle: "test-handle"}

	resp, err := client.GetPVCLabelsByPVName(context.Background(), req)
	if err == nil || !strings.Contains(err.Error(), "simulated clientset creation error") {
		t.Fatalf("expected error containing 'simulated clientset creation error', got: %v", err)
	}
	assert.Nil(t, resp)
}

func TestGetPVCLabelsByPVName_ErrorRetrievingPVC(t *testing.T) {
	// Create a PV with ClaimRef to trigger PVC retrieval
	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pv",
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeSource: v1.PersistentVolumeSource{
				CSI: &v1.CSIPersistentVolumeSource{
					VolumeHandle: "test-handle",
				},
			},
			ClaimRef: &v1.ObjectReference{
				Name:      "test-pvc",
				Namespace: "default",
			},
		},
		Status: v1.PersistentVolumeStatus{
			Phase: v1.VolumeBound,
		},
	}

	// Create fake clientset with the PV but PVC Get will fail
	fakeClientset := fake.NewSimpleClientset(pv)
	client := NewMetadataRetrieverClient(nil, 0)
	client.getClientset = func() (kubernetes.Interface, error) {
		return &pvcGetErrorClientset{Clientset: fakeClientset}, nil
	}

	req := &GetPVCLabelsByPVNameRequest{PVName: "test-pv", VolumeHandle: "test-handle"}

	resp, err := client.GetPVCLabelsByPVName(context.Background(), req)
	if err == nil || !strings.Contains(err.Error(), "simulated PVC get error") {
		t.Fatalf("expected error containing 'simulated PVC get error', got: %v", err)
	}
	if resp != nil {
		t.Fatalf("expected nil response, got: resp=%v", resp)
	}
}

func TestGetPVCLabelsByPVName_PVNotFound(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset()

	client := NewMetadataRetrieverClient(nil, 0)
	client.getClientset = func() (kubernetes.Interface, error) {
		return fakeClientset, nil
	}
	req := &GetPVCLabelsByPVNameRequest{PVName: "nonexistent-pv", VolumeHandle: "test-handle"}

	resp, err := client.GetPVCLabelsByPVName(context.Background(), req)
	if err == nil || !strings.Contains(err.Error(), "persistentvolumes \"nonexistent-pv\" not found") {
		t.Fatalf("expected PV not found error, got: %v", err)
	}
	if resp != nil {
		t.Fatalf("expected nil response, got: resp=%v", resp)
	}
}

func TestGetPVCLabelsByPVName_PVNoClaimRef(t *testing.T) {
	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pv",
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeSource: v1.PersistentVolumeSource{
				CSI: &v1.CSIPersistentVolumeSource{
					VolumeHandle: "test-handle",
				},
			},
		},
		Status: v1.PersistentVolumeStatus{
			Phase: v1.VolumeBound,
		},
	}

	fakeClientset := fake.NewSimpleClientset(pv)

	client := NewMetadataRetrieverClient(nil, 0)
	client.getClientset = func() (kubernetes.Interface, error) {
		return fakeClientset, nil
	}
	req := &GetPVCLabelsByPVNameRequest{PVName: "test-pv", VolumeHandle: "test-handle"}

	resp, err := client.GetPVCLabelsByPVName(context.Background(), req)
	if err == nil || !strings.Contains(err.Error(), "PV test-pv has no ClaimRef") {
		t.Fatalf("expected error: PV test-pv has no ClaimRef, got: %v", err)
	}
	if resp != nil {
		t.Fatalf("expected nil response, got: resp=%v", resp)
	}
}

func TestGetPVCLabelsByPVName_PVCBoundToDifferentPV(t *testing.T) {
	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pv",
		},
		Spec: v1.PersistentVolumeSpec{
			ClaimRef: &v1.ObjectReference{
				Name:      "test-pvc",
				Namespace: "default",
			},
			PersistentVolumeSource: v1.PersistentVolumeSource{
				CSI: &v1.CSIPersistentVolumeSource{
					VolumeHandle: "test-handle",
				},
			},
		},
		Status: v1.PersistentVolumeStatus{
			Phase: v1.VolumeBound,
		},
	}

	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc",
			Namespace: "default",
			Labels:    map[string]string{"key1": "value1"},
		},
		Status: v1.PersistentVolumeClaimStatus{
			Phase: v1.ClaimBound,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			VolumeName: "test-pv-not-match",
		},
	}

	fakeClientset := fake.NewSimpleClientset(pv, pvc)

	client := NewMetadataRetrieverClient(nil, 0)
	client.getClientset = func() (kubernetes.Interface, error) {
		return fakeClientset, nil
	}
	req := &GetPVCLabelsByPVNameRequest{PVName: "test-pv", VolumeHandle: "test-handle"}

	resp, err := client.GetPVCLabelsByPVName(context.Background(), req)
	if err == nil || !strings.Contains(err.Error(), "bound to a different PV") {
		t.Fatalf("expected error about PVC bound to a different PV, got: %v", err)
	}
	if resp != nil {
		t.Fatalf("expected nil response, got: resp=%v", resp)
	}
}

func TestGetPVCLabelsByPVName_VolumeHandleMismatch(t *testing.T) {
	// Create a PV with different volume handle
	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pv",
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeSource: v1.PersistentVolumeSource{
				CSI: &v1.CSIPersistentVolumeSource{
					VolumeHandle: "different-handle", // Different from request
				},
			},
			ClaimRef: &v1.ObjectReference{
				Name:      "test-pvc",
				Namespace: "default",
			},
		},
		Status: v1.PersistentVolumeStatus{
			Phase: v1.VolumeBound,
		},
	}

	fakeClientset := fake.NewSimpleClientset(pv)

	client := NewMetadataRetrieverClient(nil, 0)
	client.getClientset = func() (kubernetes.Interface, error) {
		return fakeClientset, nil
	}
	req := &GetPVCLabelsByPVNameRequest{
		PVName:       "test-pv",
		VolumeHandle: "expected-handle", // Different from PV
	}

	resp, err := client.GetPVCLabelsByPVName(context.Background(), req)
	if err == nil || !strings.Contains(err.Error(), "volume handle mismatch") {
		t.Fatalf("expected volume handle mismatch error, got: %v", err)
	}
	if resp != nil {
		t.Fatalf("expected nil response, got: resp=%v", resp)
	}
}

func TestGetPVCLabelsByPVName_VolumeHandleValidationNilCSI(t *testing.T) {
	// Create a PV with nil CSI spec
	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pv",
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeSource: v1.PersistentVolumeSource{
				CSI: nil, // CSI is nil
			},
			ClaimRef: &v1.ObjectReference{
				Name:      "test-pvc",
				Namespace: "default",
			},
		},
		Status: v1.PersistentVolumeStatus{
			Phase: v1.VolumeBound,
		},
	}

	fakeClientset := fake.NewSimpleClientset(pv)

	client := NewMetadataRetrieverClient(nil, 0)
	client.getClientset = func() (kubernetes.Interface, error) {
		return fakeClientset, nil
	}
	req := &GetPVCLabelsByPVNameRequest{
		PVName:       "test-pv",
		VolumeHandle: "expected-handle", // Provide handle but CSI is nil
	}

	resp, err := client.GetPVCLabelsByPVName(context.Background(), req)
	if err == nil || !strings.Contains(err.Error(), "volume handle mismatch") {
		t.Fatalf("expected volume handle mismatch error when CSI is nil, got: %v", err)
	}
	if resp != nil {
		t.Fatalf("expected nil response, got: resp=%v", resp)
	}
}

func TestGetPVCLabelsByPVName_PVNameOnly(t *testing.T) {
	// Test basic lookup with PV name only
	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pv"},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeSource: v1.PersistentVolumeSource{
				CSI: &v1.CSIPersistentVolumeSource{VolumeHandle: "test-handle"},
			},
			ClaimRef: &v1.ObjectReference{Name: "test-pvc", Namespace: "default"},
		},
		Status: v1.PersistentVolumeStatus{Phase: v1.VolumeBound},
	}
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pvc", Namespace: "default",
			Labels: map[string]string{"app": "myapp", "env": "prod"},
		},
		Spec:   v1.PersistentVolumeClaimSpec{VolumeName: "test-pv"},
		Status: v1.PersistentVolumeClaimStatus{Phase: v1.ClaimBound},
	}

	client := createTestClient(func() (kubernetes.Interface, error) {
		return fake.NewSimpleClientset(pv, pvc), nil
	})
	req := &GetPVCLabelsByPVNameRequest{
		PVName: "test-pv", PVCName: "", PVCNamespace: "", VolumeHandle: "test-handle",
	}

	resp, err := client.GetPVCLabelsByPVName(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, "test-pvc", resp.PVCName)
	assert.Equal(t, "default", resp.PVCNamespace)
	assert.Equal(t, "test-pv", resp.PVName)
	assert.Equal(t, map[string]string{"app": "myapp", "env": "prod"}, resp.Parameters)
}

func TestGetPVCLabelsByPVName_FastPath_AllFieldsProvided(t *testing.T) {
	// Test fast path: all fields provided and PVC exists and is bound to the PV
	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: "fast-pv"},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeSource: v1.PersistentVolumeSource{
				CSI: &v1.CSIPersistentVolumeSource{VolumeHandle: "fast-handle"},
			},
			ClaimRef: &v1.ObjectReference{Name: "fast-pvc", Namespace: "fast-ns"},
		},
		Status: v1.PersistentVolumeStatus{Phase: v1.VolumeBound},
	}
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fast-pvc", Namespace: "fast-ns",
			Labels: map[string]string{"team": "storage"},
		},
		Spec:   v1.PersistentVolumeClaimSpec{VolumeName: "fast-pv"},
		Status: v1.PersistentVolumeClaimStatus{Phase: v1.ClaimBound},
	}

	client := createTestClient(func() (kubernetes.Interface, error) {
		return fake.NewSimpleClientset(pv, pvc), nil
	})
	req := &GetPVCLabelsByPVNameRequest{
		PVName: "fast-pv", PVCName: "fast-pvc", PVCNamespace: "fast-ns", VolumeHandle: "fast-handle",
	}

	resp, err := client.GetPVCLabelsByPVName(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, "fast-pvc", resp.PVCName)
	assert.Equal(t, "fast-ns", resp.PVCNamespace)
	assert.Equal(t, "fast-pv", resp.PVName)
	assert.Equal(t, map[string]string{"team": "storage"}, resp.Parameters)
}

func TestGetPVCLabelsByPVName_VolumeHandleOnly(t *testing.T) {
	// Test volume handle only lookup via PV list
	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: "list-pv"},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeSource: v1.PersistentVolumeSource{
				CSI: &v1.CSIPersistentVolumeSource{VolumeHandle: "list-handle"},
			},
			ClaimRef: &v1.ObjectReference{Name: "list-pvc", Namespace: "list-ns"},
		},
		Status: v1.PersistentVolumeStatus{Phase: v1.VolumeBound},
	}
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "list-pvc", Namespace: "list-ns",
			Labels: map[string]string{"type": "list"},
		},
		Spec:   v1.PersistentVolumeClaimSpec{VolumeName: "list-pv"},
		Status: v1.PersistentVolumeClaimStatus{Phase: v1.ClaimBound},
	}

	client := createTestClient(func() (kubernetes.Interface, error) {
		return fake.NewSimpleClientset(pv, pvc), nil
	})
	req := &GetPVCLabelsByPVNameRequest{
		PVName: "", PVCName: "", PVCNamespace: "", VolumeHandle: "list-handle",
	}

	resp, err := client.GetPVCLabelsByPVName(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, "list-pvc", resp.PVCName)
	assert.Equal(t, "list-ns", resp.PVCNamespace)
	assert.Equal(t, "list-pv", resp.PVName)
	assert.Equal(t, map[string]string{"type": "list"}, resp.Parameters)
}

func TestGetPVCLabelsByPVName_PVNotBound(t *testing.T) {
	// PV exists but is not bound — should fail
	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: "pv-avail"},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeSource: v1.PersistentVolumeSource{
				CSI: &v1.CSIPersistentVolumeSource{VolumeHandle: "vh-avail"},
			},
		},
		Status: v1.PersistentVolumeStatus{Phase: v1.VolumeAvailable},
	}

	client := createTestClient(func() (kubernetes.Interface, error) {
		return fake.NewSimpleClientset(pv), nil
	})
	req := &GetPVCLabelsByPVNameRequest{PVName: "pv-avail", VolumeHandle: "vh-avail"}

	resp, err := client.GetPVCLabelsByPVName(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "PV is not bound")
	assert.Nil(t, resp)
}

func TestGetPVCLabelsByPVName_PVListError(t *testing.T) {
	// Volume handle lookup path — PV list returns error
	client := createTestClient(FakeGetClientsetPVListError)
	req := &GetPVCLabelsByPVNameRequest{PVName: "", VolumeHandle: "some-handle"}

	resp, err := client.GetPVCLabelsByPVName(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "simulated PV list error")
	assert.Nil(t, resp)
}

func TestGetPVCLabelsByPVName_VolumeHandleNotFoundInList(t *testing.T) {
	// No PV in the list matches the requested volume handle
	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: "pv-other"},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeSource: v1.PersistentVolumeSource{
				CSI: &v1.CSIPersistentVolumeSource{VolumeHandle: "other-handle"},
			},
		},
		Status: v1.PersistentVolumeStatus{Phase: v1.VolumeBound},
	}

	client := createTestClient(func() (kubernetes.Interface, error) {
		return fake.NewSimpleClientset(pv), nil
	})
	req := &GetPVCLabelsByPVNameRequest{PVName: "", VolumeHandle: "missing-handle"}

	resp, err := client.GetPVCLabelsByPVName(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "PV with VolumeHandle=missing-handle not found")
	assert.Nil(t, resp)
}

func TestGetPVCLabelsByPVName_PVCNotBound(t *testing.T) {
	// PV is valid but the bound PVC is in Pending state
	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: "pv-nb"},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeSource: v1.PersistentVolumeSource{
				CSI: &v1.CSIPersistentVolumeSource{VolumeHandle: "vh-nb"},
			},
			ClaimRef: &v1.ObjectReference{Name: "pvc-nb", Namespace: "default"},
		},
		Status: v1.PersistentVolumeStatus{Phase: v1.VolumeBound},
	}
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "pvc-nb", Namespace: "default"},
		Spec:       v1.PersistentVolumeClaimSpec{VolumeName: "pv-nb"},
		Status:     v1.PersistentVolumeClaimStatus{Phase: v1.ClaimPending},
	}

	client := createTestClient(func() (kubernetes.Interface, error) {
		return fake.NewSimpleClientset(pv, pvc), nil
	})
	req := &GetPVCLabelsByPVNameRequest{PVName: "pv-nb", VolumeHandle: "vh-nb"}

	resp, err := client.GetPVCLabelsByPVName(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "PVC is not bound")
	assert.Nil(t, resp)
}

func TestGetPVCLabelsByPVName_NilLabels(t *testing.T) {
	// PVC has no labels at all — should return empty map (nil)
	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: "pv-nl"},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeSource: v1.PersistentVolumeSource{
				CSI: &v1.CSIPersistentVolumeSource{VolumeHandle: "vh-nl"},
			},
			ClaimRef: &v1.ObjectReference{Name: "pvc-nl", Namespace: "default"},
		},
		Status: v1.PersistentVolumeStatus{Phase: v1.VolumeBound},
	}
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "pvc-nl", Namespace: "default"},
		Spec:       v1.PersistentVolumeClaimSpec{VolumeName: "pv-nl"},
		Status:     v1.PersistentVolumeClaimStatus{Phase: v1.ClaimBound},
	}

	client := createTestClient(func() (kubernetes.Interface, error) {
		return fake.NewSimpleClientset(pv, pvc), nil
	})
	req := &GetPVCLabelsByPVNameRequest{PVName: "pv-nl", VolumeHandle: "vh-nl"}

	resp, err := client.GetPVCLabelsByPVName(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, "pvc-nl", resp.PVCName)
	assert.Equal(t, "default", resp.PVCNamespace)
	assert.Equal(t, "pv-nl", resp.PVName)
	assert.Empty(t, resp.Parameters)
}
