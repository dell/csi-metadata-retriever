package utils

import (
	"errors"
	"net"
	"os"

	"github.com/dell/csi-metadata-retriever/retriever"
)

// GetCSIEndpoint returns the network address specified by the
// environment variable CSI_RETRIEVER_ENDPOINT.
func GetCSIEndpoint() (network, addr string, err error) {
	protoAddr := os.Getenv(retriever.EnvVarEndpoint)
	if emptyRX.MatchString(protoAddr) {
		return "", "", errors.New("missing CSI_RETRIEVER_ENDPOINT")
	}
	return ParseProtoAddr(protoAddr)
}

// GetCSIEndpointListener returns the net.Listener for the endpoint
// specified by the environment variable CSI_ENDPOINT.
func GetCSIEndpointListener() (net.Listener, error) {
	proto, addr, err := GetCSIEndpoint()
	if err != nil {
		return nil, err
	}
	return net.Listen(proto, addr)
}
