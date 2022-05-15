package service

import (
	"github.com/container-storage-interface/spec/lib/go/csi"
)

const (
	// Name is the name of this CSI SP.
	Name = "newcsi"

	// VendorVersion is the version of this CSP SP.
	VendorVersion = "0.0.0"
)

// Service is a CSI SP and idempotency.Provider.
type Service interface {
	csi.ControllerServer
	csi.IdentityServer
	csi.NodeServer
}

type service struct{}

// New returns a new Service.
func New() Service {
	return &service{}
}
