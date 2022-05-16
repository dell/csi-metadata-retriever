package service

import (
	"errors"
	"fmt"

	"github.com/dell/csi-metadata-retriever/retriever"
	"golang.org/x/net/context"

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

	_ = clientset.CoreV1().PersistentVolumes()
	//match result with req.Name
	if err != nil {
		panic(err.Error())
	}

	resp := &retriever.GetPVCLabelsResponse{}
	return resp, err
}
