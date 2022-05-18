package utils

import (
	"errors"
	"net"
	"os"
	"regexp"

	gocsiutils "github.com/dell/gocsi/utils"
)

var (
	emptyRX = regexp.MustCompile(`^\s*$`)
)

// GetCSIEndpoint returns the network address specified by the
// environment variable CSI_RETRIEVER_ENDPOINT.
func GetCSIEndpoint() (network, addr string, err error) {
	protoAddr := os.Getenv(EnvVarEndpoint)
	if emptyRX.MatchString(protoAddr) {
		return "", "", errors.New("missing CSI_RETRIEVER_ENDPOINT")
	}
	return gocsiutils.ParseProtoAddr(protoAddr)
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
