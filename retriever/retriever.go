/*
 *
 * Copyright © 2021-2024 Dell Inc. or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

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
	"errors"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	log "github.com/sirupsen/logrus"
)

var restInClusterConfig = rest.InClusterConfig

// MetadataRetrieverClient is the interface for retrieving metadata.
type MetadataRetrieverClient interface {
	GetPVCLabels(context.Context, *GetPVCLabelsRequest) (*GetPVCLabelsResponse, error)
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
