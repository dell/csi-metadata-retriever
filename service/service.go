package service

import (
	"github.com/dell/csi-metadata-retriever/retriever"
)

const (
	// Name is the name of this CSI SP.
	Name = "newcsi"

	// VendorVersion is the version of this CSP SP.
	VendorVersion = "0.0.0"
)

// Service is a CSI SP and idempotency.Provider.
type Service interface {
	retriever.RetrieverServer
}

type service struct{}

// New returns a new Service.
func New() Service {
	return &service{}
}
