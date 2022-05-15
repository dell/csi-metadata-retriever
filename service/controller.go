package retriever

import (
	"golang.org/x/net/context"

	"github.com/container-storage-interface/spec/lib/go/csi"
)

func (s *service) GetPVCLabels(
	ctx context.Context,
	req *csi.GetPVCLabelsRequest) (
	*csi.GetPVCLabelsResponse, error) {

	return nil, nil
}
