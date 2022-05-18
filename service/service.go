package service

const (
	// Name is the name of this CSI SP.
	Name = "csi-metadata-retriever"

	// VendorVersion is the version of this CSP SP.
	VendorVersion = "0.0.0"
)

// Service is a CSI SP and idempotency.Provider.
type Service interface {
	//	retriever.MetadataRetrieverClient
}

type service struct{}

// New returns a new Service.
func New() Service {
	return &service{}
}
