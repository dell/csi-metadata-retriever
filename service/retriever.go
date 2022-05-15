package service

import (
	"github.com/dell/csi-metadata-retriever/retriever"
	"golang.org/x/net/context"
)

func (s *service) GetPVCLabels(
	ctx context.Context,
	req *retriever.GetPVCLabelsRequest) (
	*retriever.GetPVCLabelsResponse, error) {

	return nil, nil
}
