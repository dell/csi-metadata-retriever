/*
 *
 * Copyright © 2025 Dell Inc. or its subsidiaries. All Rights Reserved.
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

package mocks

import (
	"context"
	"errors"
	"net"
	"os"
	"os/user"

	"github.com/dell/csi-metadata-retriever/service"
	"github.com/stretchr/testify/mock"
)

// MockListener mocks a net.Listener for testing.
type MockListener struct {
	mock.Mock
}

func (m *MockListener) Accept() (net.Conn, error) {
	return nil, errors.New("mock accept error")
}

func (m *MockListener) Close() error {
	return nil
}

func (m *MockListener) Addr() net.Addr {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(net.Addr)
}

// MockAddr mocks a net.Addr for testing.
type MockAddr struct {
	NetworkField string
	AddressField string
}

func (m *MockAddr) Network() string {
	return m.NetworkField
}

func (m *MockAddr) String() string {
	return m.AddressField
}

// MockPluginProvider mocks the PluginProvider interface for testing.
type MockPluginProvider struct {
	mock.Mock
}

func (m *MockPluginProvider) Serve(ctx context.Context, l net.Listener) error {
	args := m.Called(ctx, l)
	return args.Error(0)
}

func (m *MockPluginProvider) GracefulStop(ctx context.Context) {
	m.Called(ctx)
}

func (m *MockPluginProvider) Stop(ctx context.Context) {
	m.Called(ctx)
}

// MockService mocks a service.Service for testing.
type MockService struct {
	service.Service
	mock.Mock
}

type MockOS struct {
	mock.Mock
}

func (m *MockOS) Chown(name string, uid, gid int) error {
	args := m.Called(name, uid, gid)
	return args.Error(0)
}

func (m *MockOS) Chmod(address string, mode os.FileMode) error {
	args := m.Called(address, mode)
	return args.Error(0)
}

type MockUser struct {
	mock.Mock
}

func (m *MockUser) LookupId(id string) (*user.User, error) {
	args := m.Called(id)
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUser) LookupGroupId(id string) (*user.Group, error) {
	args := m.Called(id)
	return args.Get(0).(*user.Group), args.Error(1)
}
