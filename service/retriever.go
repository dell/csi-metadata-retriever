package service

import (
	"errors"
	"fmt"

	"github.com/dell/csi-metadata-retriever/retriever"
	"golang.org/x/net/context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func (s *service) GetPVCLabels(
	ctx context.Context,
	req *retriever.GetPVCLabelsRequest) (
	*retriever.GetPVCLabelsResponse, error) {

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

	resp := &retriever.GetPVCLabelsResponse{}

	for k, v := range pvc.Labels {
		resp.Parameters[k] = v
	}

	return resp, err
}
