package retriever

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

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

type MetadataRetrieverClientType struct {
	conn *grpc.ClientConn
	//log     logr.Logger
	timeout time.Duration
}

//NewMetadataRetrieverClient returns csiclient
func NewMetadataRetrieverClient(conn *grpc.ClientConn, timeout time.Duration) *MetadataRetrieverClientType {
	return &MetadataRetrieverClientType{
		conn: conn,
		//log:     log,
		timeout: timeout,
	}
}

func (s *MetadataRetrieverClientType) GetPVCLabels(
	ctx context.Context,
	req *GetPVCLabelsRequest) (
	*GetPVCLabelsResponse, error) {

	fmt.Print("----- Inside Get PVC Labels RPC -----")

	if req.Name == "" {
		return nil, errors.New(
			"PVC Name cannot be empty")
	}

	//TODO: config and clientset to be moved to BeforeServe()
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	pvcClient := clientset.CoreV1().PersistentVolumeClaims(req.NameSpace)
	if pvcClient == nil {
		panic(errors.New("PVC client is nil"))
	}

	pvc, err := pvcClient.Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		panic(err.Error())
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
