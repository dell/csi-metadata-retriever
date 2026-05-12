/*
 *
 * Copyright © 2022-2025 Dell Inc. or its subsidiaries. All Rights Reserved.
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
	"fmt"
	"time"

	"google.golang.org/grpc"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	log "github.com/sirupsen/logrus"
)

var restInClusterConfig = rest.InClusterConfig

// MetadataRetrieverClient is the interface for retrieving metadata.
type MetadataRetrieverClient interface {
	GetPVCLabels(context.Context, *GetPVCLabelsRequest) (*GetPVCLabelsResponse, error)
	GetPVCLabelsByPVName(context.Context, *GetPVCLabelsByPVNameRequest) (*GetPVCLabelsByPVNameResponse, error)
}

// GetPVCLabelsRequest defines API request type
type GetPVCLabelsRequest struct {
	Name      string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	NameSpace string `protobuf:"bytes,1,opt,name=namespace,proto3" json:"namespace,omitempty"`
}

// GetPVCLabelsResponse defines API response type
type GetPVCLabelsResponse struct {
	Parameters map[string]string `protobuf:"bytes,4,rep,name=parameters,proto3" json:"parameters,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

type GetPVCLabelsByPVNameRequest struct {
	// Optional PV name, if empty - the method will fall back to listing all PVs and matching by volume handle
	PVName string `protobuf:"bytes,2,opt,name=pv_name,proto3" json:"pv_name,omitempty"`
	// Optional PVC name, if provided - it is used to query PVC resource directly
	PVCName string `protobuf:"bytes,1,opt,name=pvc_name,proto3" json:"pvc_name,omitempty"`
	// Optional PVC namespace, if provided - it is used to query PVC resource directly
	PVCNamespace string `protobuf:"bytes,1,opt,name=pvc_namespace,proto3" json:"pvc_namespace,omitempty"`
	// Required volume handle, it is used to match the PV or validate the looked up PVC
	VolumeHandle string `protobuf:"bytes,3,opt,name=volume_handle,proto3" json:"volume_handle,omitempty"`
}

type GetPVCLabelsByPVNameResponse struct {
	PVName       string            `protobuf:"bytes,1,name=pv_name,proto3" json:"pv_name,omitempty"`
	PVCName      string            `protobuf:"bytes,2,name=pvc_name,proto3" json:"pvc_name,omitempty"`
	PVCNamespace string            `protobuf:"bytes,3,name=pvc_namespace,proto3" json:"pvc_namespace,omitempty"`
	Parameters   map[string]string `protobuf:"bytes,4,rep,name=parameters,proto3" json:"parameters,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

// MetadataRetrieverClientType holds client connection and timeout
type MetadataRetrieverClientType struct {
	conn         *grpc.ClientConn
	timeout      time.Duration
	getClientset func() (kubernetes.Interface, error)
}

// NewMetadataRetrieverClient returns csiclient
func NewMetadataRetrieverClient(conn *grpc.ClientConn, timeout time.Duration) *MetadataRetrieverClientType {
	return &MetadataRetrieverClientType{
		conn:         conn,
		timeout:      timeout,
		getClientset: defaultGetClientset,
	}
}

func defaultGetClientset() (kubernetes.Interface, error) {
	config, err := restInClusterConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

// GetPVCLabels gets the PVC labels and returns it
func (s *MetadataRetrieverClientType) GetPVCLabels(
	ctx context.Context,
	req *GetPVCLabelsRequest) (
	*GetPVCLabelsResponse, error,
) {
	log.Infof("Get PVC labels for %s in namespace %s", req.Name, req.NameSpace)
	if req.Name == "" {
		return nil, errors.New(
			"PVC Name cannot be empty")
	}

	clientset, err := s.getClientset()
	if err != nil {
		log.Error("Error creating clientset: ", err)
		return nil, err
	}

	pvcClient := clientset.CoreV1().PersistentVolumeClaims(req.NameSpace)
	if pvcClient == nil {
		log.Error("Error getting PVC client: ", err)
		return nil, err
	}

	pvc, err := pvcClient.Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		log.Error("Error retrieving PVC info: ", err)
		return nil, err
	}

	parameters := make(map[string]string)

	for k, v := range pvc.Labels {
		parameters[k] = v
	}

	resp := &GetPVCLabelsResponse{
		Parameters: parameters,
	}

	return resp, err
}

// GetPVCLabelsByPVName gets the PVC labels by PV name directly
func (s *MetadataRetrieverClientType) GetPVCLabelsByPVName(
	ctx context.Context,
	req *GetPVCLabelsByPVNameRequest,
) (*GetPVCLabelsByPVNameResponse, error) {
	volumeHandle := req.VolumeHandle
	pvName := req.PVName
	pvcName := req.PVCName
	pvcNamespace := req.PVCNamespace

	log.Debugf("Getting PVC labels by VolumeHandle=%s, PVName=%s, PVCName=%s, PVCNamespace=%s",
		volumeHandle, pvName, pvcName, pvcNamespace)

	if volumeHandle == "" {
		return nil, errors.New("VolumeHandle is empty")
	}

	clientset, err := s.getClientset()
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %v", err)
	}

	var pvc *v1.PersistentVolumeClaim

	// The fast path - PVC name and namespace is specified and they are not stale
	if pvcName != "" && pvcNamespace != "" && pvName != "" {
		pvc, err = s.lookupPVC(ctx, clientset, pvcName, pvcNamespace, pvName)
		if err != nil {
			// Assuming the PVC with this name has been removed or has been re-bound to another PV
			log.Debugf("Could not retrieve the PVC %s/%s: %v", pvcNamespace, pvcName, err)
		}
	}

	if pvc == nil {
		// Lookup PV first and then get the bound PVC
		pv, err := s.lookupPV(ctx, clientset, pvName, volumeHandle)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve PV: %v", err)
		}
		// Save pvName if it was not specified
		pvName = pv.Name
		// Get PVC name from PV's ClaimRef
		if pv.Spec.ClaimRef == nil {
			return nil, fmt.Errorf("PV %s has no ClaimRef", pvName)
		}
		pvcName = pv.Spec.ClaimRef.Name
		pvcNamespace = pv.Spec.ClaimRef.Namespace
		// Finally retrieve the PVC
		pvc, err = s.lookupPVC(ctx, clientset, pvcName, pvcNamespace, pvName)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve bound PVC %s/%s: %v", pvcNamespace, pvcName, err)
		}
	}

	log.Debugf("Retrieving labels for PVC %s/%s", pvcNamespace, pvcName)

	return &GetPVCLabelsByPVNameResponse{
		Parameters:   pvc.Labels,
		PVName:       pvName,
		PVCName:      pvcName,
		PVCNamespace: pvcNamespace,
	}, nil
}

// lookupPV finds a PV by name or by volume handle if name is empty
func (s *MetadataRetrieverClientType) lookupPV(ctx context.Context, clientset kubernetes.Interface, pvName, volumeHandle string) (*v1.PersistentVolume, error) {
	pvClient := clientset.CoreV1().PersistentVolumes()

	if pvName != "" {
		// Lookup by name
		pv, err := pvClient.Get(ctx, pvName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get PV %s: %v", pvName, err)
		}

		if pv.Status.Phase != v1.VolumeBound {
			return nil, fmt.Errorf("PV is not bound")
		}

		// Validate volume handle matches
		if pv.Spec.CSI == nil || pv.Spec.CSI.VolumeHandle != volumeHandle {
			return nil, fmt.Errorf("PV %s volume handle mismatch: expected %s, got %v",
				pvName, volumeHandle, pv.Spec.CSI)
		}

		return pv, nil
	}

	// Lookup by volume handle - list all PVs and find matching one
	pvList, err := pvClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list PVs: %v", err)
	}

	for i := range pvList.Items {
		pv := pvList.Items[i]
		if pv.Spec.CSI != nil && pv.Spec.CSI.VolumeHandle == volumeHandle && pv.Status.Phase == v1.VolumeBound {
			return &pv, nil
		}
	}

	return nil, fmt.Errorf("PV with VolumeHandle=%s not found", volumeHandle)
}

// lookupPVC finds a PVC by name and namespace and validates it's bound
func (s *MetadataRetrieverClientType) lookupPVC(ctx context.Context, clientset kubernetes.Interface,
	pvcName, pvcNamespace, pvName string,
) (*v1.PersistentVolumeClaim, error) {
	pvcClient := clientset.CoreV1().PersistentVolumeClaims(pvcNamespace)

	pvc, err := pvcClient.Get(ctx, pvcName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve get PVC by name: %v", err)
	}

	if pvc.Status.Phase != v1.ClaimBound {
		return nil, fmt.Errorf("PVC is not bound")
	}

	// Such a PVC exist and it's bound, but what if it has been re-created and is now bound to another PV.
	if pvc.Spec.VolumeName != pvName {
		return nil, fmt.Errorf("PVC found, but it is bound to a different PV %s", pvc.Spec.VolumeName)
	}

	return pvc, nil
}
