/*
 *
 * Copyright © 2022-2025 Dell Inc. or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *      http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package csiendpoint

import (
	"errors"
	"net"
	"os"
	"regexp"

	gocsiutils "github.com/dell/gocsi/utils/csi"
)

var emptyRX = regexp.MustCompile(`^\s*$`)

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
// For Unix sockets, any stale socket file from a previous instance is
// removed before listening to prevent "address already in use" errors
// after non-graceful termination (e.g. node reboot where SIGKILL
// bypasses the signal handler).
func GetCSIEndpointListener() (net.Listener, error) {
	proto, addr, err := GetCSIEndpoint()
	if err != nil {
		return nil, err
	}
	if proto == "unix" {
		/* #nosec G104 */ // best-effort cleanup; net.Listen will fail if path is still unusable
		os.Remove(addr)
	}
	return net.Listen(proto, addr)
}
