package provider

import (
	"context"
	"net"

	"github.com/dell/csi-metadata-retriever/retriever"
	"github.com/dell/csi-metadata-retriever/service"
	"github.com/dell/gocsi"
	log "github.com/sirupsen/logrus"
)

// New returns a new CSI Storage Plug-in Provider.
func New() retriever.RetrieverPluginProvider {
	svc := service.New()
	return &retriever.RetrieverPlugin{
		MetadataRetriever: svc,

		// BeforeServe allows the SP to participate in the startup
		// sequence. This function is invoked directly before the
		// gRPC server is created, giving the callback the ability to
		// modify the SP's interceptors, server options, or prevent the
		// server from starting by returning a non-nil error.
		BeforeServe: func(
			ctx context.Context,
			sp *retriever.RetrieverPlugin,
			lis net.Listener) error {

			log.WithField("service", "MetadataRetriever").Debug("BeforeServe")
			return nil
		},

		EnvVars: []string{
			// Enable request validation.
			gocsi.EnvVarSpecReqValidation + "=true",

			// Enable serial volume access.
			gocsi.EnvVarSerialVolAccess + "=true",
		},
	}
}
