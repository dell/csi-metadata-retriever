/*
 *
 * Copyright © 2022 Dell Inc. or its subsidiaries. All Rights Reserved.
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

package retriever

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/dell/csi-metadata-retriever/service"
	"github.com/dell/csi-metadata-retriever/utils"
	"github.com/dell/gocsi"
	csictx "github.com/dell/gocsi/context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// MockListener mocks a net.Listener for testing.
type MockListener struct {
	net.Listener
	// addr net.Addr
}

func (m *MockListener) Accept() (net.Conn, error) {
	return nil, errors.New("mock accept error")
}

func (m *MockListener) Close() error {
	return nil
}

func (m *MockListener) Addr() net.Addr {
	return &MockAddr{network: "unix", address: "/tmp/mock.sock"}
	// return m.addr
}

// MockAddr mocks a net.Addr for testing.
type MockAddr struct {
	network string
	address string
}

func (m *MockAddr) Network() string {
	return m.network
}

func (m *MockAddr) String() string {
	return m.address
}

// MockService mocks a service.Service for testing.
type MockService struct {
	service.Service
	mock.Mock
}

var grpcClient *grpc.ClientConn

func TestServer_StartGracefulStop(_ *testing.T) {
	// var stop func()
	os.Setenv("CSI_RETRIEVER_ENDPOINT", "/tmp/csi_retriever_test.sock")

	ctx := context.Background()
	sp := new(Plugin)
	sp.MetadataRetrieverService = service.New()

	fmt.Printf("calling startServer")
	grpcClient, _ = startServer(ctx, sp, true)
	fmt.Printf("back from startServer")
	time.Sleep(5 * time.Second)

	// stop()
}

func TestServer_StartStop(_ *testing.T) {
	// var stop func()
	os.Setenv("CSI_RETRIEVER_ENDPOINT", "/tmp/csi_retriever_test.sock")

	ctx := context.Background()
	sp := new(Plugin)
	sp.MetadataRetrieverService = service.New()

	fmt.Printf("calling startServer")
	grpcClient, _ = startServer(ctx, sp, false)
	fmt.Printf("back from startServer")
	time.Sleep(5 * time.Second)

	// stop()
}

func startServer(ctx context.Context, sp *Plugin, gracefulStop bool) (*grpc.ClientConn, func()) {
	lis, err := utils.GetCSIEndpointListener()
	if err != nil {
		fmt.Printf("couldn't open listener: %s\n", err.Error())
		return nil, nil
	}

	fmt.Printf("lis: %v\n", lis)
	go func() {
		fmt.Printf("starting server\n")
		if err := sp.Serve(ctx, lis); err != nil {
			fmt.Printf("http: Server closed. Error: %v", err)
		}
	}()
	network, addr, err := utils.GetCSIEndpoint()
	if err != nil {
		return nil, nil
	}
	fmt.Printf("network %v addr %v\n", network, addr)

	clientOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	// Create a client for the piped connection.
	fmt.Printf("calling gprc.DialContext, ctx %v, addr %s, clientOpts %v\n", ctx, addr, clientOpts)
	client, err := grpc.DialContext(ctx, "unix:"+addr, clientOpts...)
	if err != nil {
		fmt.Printf("DialContext returned error: %s", err.Error())
	}
	fmt.Printf("grpc.DialContext returned ok\n")

	if gracefulStop {
		return client, func() {
			client.Close()
			sp.GracefulStop(ctx)
		}
	}

	return client, func() {
		client.Close()
		sp.Stop(ctx)
	}
}

func TestPlugin_initEndpointPerms(t *testing.T) {
	// Mock os.Chmod to avoid actual filesystem changes
	monkey.Patch(os.Chmod, func(name string, mode os.FileMode) error {
		return nil
	})
	defer monkey.Unpatch(os.Chmod)

	tests := []struct {
		name        string
		plugin      *Plugin
		envVarValue string
		network     string
		expectedErr error
	}{
		{
			name: "Default Permission Value",
			plugin: &Plugin{
				EnvVars: []string{},
			},
			envVarValue: "0755",
			network:     netUnix,
			expectedErr: nil,
		},
		{
			name: "Non-Unix Network",
			plugin: &Plugin{
				EnvVars: []string{},
			},
			envVarValue: "0755",
			network:     "tcp",
			expectedErr: nil,
		},
		{
			name: "Invalid Permission Value",
			plugin: &Plugin{
				EnvVars: []string{},
			},
			envVarValue: "invalid",
			network:     netUnix,
			expectedErr: &strconv.NumError{},
		},
		{
			name: "Chmod Error",
			plugin: &Plugin{
				EnvVars: []string{},
			},
			envVarValue: "0777",
			network:     netUnix,
			expectedErr: &fs.PathError{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := csictx.WithLookupEnv(context.Background(), func(key string) (string, bool) {
				if key == gocsi.EnvVarEndpointPerms {
					return tt.envVarValue, true
				}
				return "", false
			})

			lis := &MockListener{}

			if tt.name == "Chmod Error" {
				monkey.Patch(os.Chmod, func(name string, mode os.FileMode) error {
					return errors.New("chmod error")
				})
				defer monkey.Unpatch(os.Chmod)
			}

			err := tt.plugin.initEndpointPerms(ctx, lis)
			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.IsType(t, tt.expectedErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPlugin_initEndpointOwner(t *testing.T) {
	monkey.Patch(os.Chown, func(name string, uid, gid int) error {
		return nil
	})
	defer monkey.Unpatch(os.Chown)

	tests := []struct {
		name        string
		plugin      *Plugin
		uid         string
		gid         string
		expectedErr bool
	}{
		{
			name: "Successful Chown",
			plugin: &Plugin{
				EnvVars: []string{},
			},
			uid:         "1000",
			gid:         "1000",
			expectedErr: false,
		},
		{
			name: "Invalid UID Format",
			plugin: &Plugin{
				EnvVars: []string{},
			},
			uid:         "invalid",
			expectedErr: true,
		},
		{
			name: "Invalid GID Format",
			plugin: &Plugin{
				EnvVars: []string{},
			},
			gid:         "invalid",
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := csictx.WithLookupEnv(context.Background(), func(key string) (string, bool) {
				switch key {
				case gocsi.EnvVarEndpointUser:
					return tt.uid, true
				case gocsi.EnvVarEndpointGroup:
					return tt.gid, true
				default:
					return "", false
				}
			})

			lis := &MockListener{}
			err := tt.plugin.initEndpointOwner(ctx, lis)
			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPlugin_lookupEnv(t *testing.T) {
	plugin := &Plugin{
		envVars: map[string]string{
			"KEY": "value",
		},
	}
	val, ok := plugin.lookupEnv("KEY")
	assert.True(t, ok)
	assert.Equal(t, "value", val)
}

func TestPlugin_setenv(t *testing.T) {
	plugin := &Plugin{
		envVars: map[string]string{},
	}
	err := plugin.setenv("KEY", "value")
	assert.NoError(t, err)
	assert.Equal(t, "value", plugin.envVars["KEY"])
}

func TestPlugin_initEnvVars(t *testing.T) {
	os.Setenv(gocsi.EnvVarDebug, "true")
	defer os.Unsetenv(gocsi.EnvVarDebug)
	tests := []struct {
		name               string
		envVars            []string
		debugEnabled       bool
		expectedEnvVars    map[string]string
		expectDebugLogging bool
	}{
		{
			name: "Normal environment variables",
			envVars: []string{
				"KEY1=value1",
				"KEY2=value2",
			},
		},
		{
			name:         "Debug environment variable enabled",
			envVars:      []string{},
			debugEnabled: true,
			expectedEnvVars: map[string]string{
				gocsi.EnvVarReqLogging: "true",
				gocsi.EnvVarRepLogging: "true",
			},
			expectDebugLogging: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := &Plugin{
				EnvVars: tt.envVars,
				envVars: map[string]string{},
			}

			ctx := context.Background()
			if tt.debugEnabled {
				ctx = csictx.WithLookupEnv(ctx, func(key string) (string, bool) {
					if key == gocsi.EnvVarDebug {
						return strconv.FormatBool(true), true
					}
					return "", false
				})
				ctx = csictx.WithSetenv(ctx, func(key, value string) error {
					plugin.envVars[key] = value
					return nil
				})
			}

			plugin.initEnvVars(ctx)

			for k, v := range tt.expectedEnvVars {
				val, ok := plugin.envVars[k]
				assert.True(t, ok, "Expected key %s to be present in envVars", k)
				assert.Equal(t, v, val)
			}
		})
	}
}

func TestPlugin_GracefulStop(t *testing.T) {
	sp := &Plugin{
		server: grpc.NewServer(),
	}
	sp.GracefulStop(context.Background())
}

func TestStop(t *testing.T) {
	sp := &Plugin{
		server: grpc.NewServer(),
	}

	sp.Stop(context.Background())
}

func TestServe(t *testing.T) {
	sp := &Plugin{
		server: grpc.NewServer(),
	}
	lis := &MockListener{}

	err := sp.Serve(context.Background(), lis)
	assert.Error(t, err)
}
