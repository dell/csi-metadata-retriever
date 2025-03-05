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

package main

import (
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/dell/csi-metadata-retriever/retriever/mocks"
	"github.com/dell/gocsi"
	csictx "github.com/dell/gocsi/context"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
)

func TestIsExitSignal(t *testing.T) {
	tests := []struct {
		name     string
		signal   os.Signal
		expected bool
	}{
		{
			name:     "SIGINT",
			signal:   os.Interrupt,
			expected: true,
		},
		{
			name:     "SIGTERM",
			signal:   os.Kill,
			expected: false,
		},
		{
			name:     "SIGHUP",
			signal:   syscall.Signal(1),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result, _ := isExitSignal(tt.signal); result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, !tt.expected)
			}
		})
	}
}

func TestTrapSignals(t *testing.T) {
	var mu sync.Mutex

	// Mock exit function
	exit = func(code int) {}

	tests := []struct {
		signal os.Signal
		exit   bool
		abort  bool
	}{
		{syscall.SIGQUIT, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.signal.String(), func(t *testing.T) {
			mu.Lock()
			exitCalled := false
			mu.Unlock()

			sigc := make(chan os.Signal, 1)
			signal.Notify(sigc, tt.signal)
			onExit := func() {
				mu.Lock()
				exitCalled = true
				mu.Unlock()
			}
			go trapSignals(onExit)

			// Send the signal
			syscall.Kill(syscall.Getpid(), tt.signal.(syscall.Signal))

			// Give some time for the signal to be processed
			time.Sleep(1 * time.Second)
			mu.Lock()
			if exitCalled != tt.exit {
				t.Errorf("expected exitCalled to be %v, got %v", tt.exit, exitCalled)
			}
			mu.Unlock()
		})
	}
}

func setEnvs(t *testing.T) {
	temp := t.TempDir()
	os.Setenv("CSI_RETRIEVER_ENDPOINT", temp+"/metadata")
	os.Setenv("X_CSI_ENDPOINT_PERMS", "0777")
	os.Setenv("X_CSI_ENDPOINT_USER", "root")
	os.Setenv("X_CSI_ENDPOINT_GROUP", "root")
	os.Setenv("X_CSI_DEBUG", "true")
	os.Setenv("X_CSI_LOG_LEVEL", "debug")
	os.Setenv("X_CSI_PLUGIN_INFO", "my-plugin")
	os.Setenv("X_CSI_REQ_LOGGING", "true")
	os.Setenv("X_CSI_REP_LOGGING", "true")
	os.Setenv("X_CSI_REQ_ID_INJECTION", "true")
	os.Setenv("X_CSI_SPEC_VALIDATION", "true")
	os.Setenv("X_CSI_SPEC_REQ_VALIDATION", "true")
	os.Setenv("X_CSI_SPEC_REP_VALIDATION", "true")
	os.Setenv("X_CSI_SPEC_DISABLE_LEN_CHECK", "true")
}

// Mock implementation of csictx.Setenv
var (
	originalSetenv = csictx.Setenv
	mockSetenv     = func(ctx context.Context, key, value string) error {
		if key == gocsi.EnvVarReqLogging {
			return errors.New("mock error")
		}
		return originalSetenv(ctx, key, value)
	}
)

func TestRun(t *testing.T) {
	var appName, appDescription, appUsage string
	ctx := context.Background()
	setEnvs(t)

	// Mock the PluginProvider
	mockProvider := new(mocks.MockPluginProvider)
	mockProvider.On("Serve", mock.Anything, mock.Anything).Return(nil)
	mockProvider.On("GracefulStop", mock.Anything).Return()
	mockProvider.On("Stop", mock.Anything).Return()

	// Override the getCSIEndpointListener variable
	getCSIEndpointListener = func() (net.Listener, error) {
		return &mocks.MockListener{}, nil
	}

	// Run the function
	Run(ctx, appName, appDescription, appUsage, mockProvider)

	// Verify the Serve method was called
	mockProvider.AssertCalled(t, "Serve", mock.Anything, mock.Anything)

	// Test case: help flag
	t.Run("help flag", func(t *testing.T) {
		os.Args = []string{"cmd", "-?"}
		Run(ctx, appName, appDescription, appUsage, mockProvider)
		// No panic or error expected
	})

	// Test case: no endpoint set
	t.Run("no endpoint set", func(t *testing.T) {
		os.Unsetenv("CSI_RETRIEVER_ENDPOINT")
		Run(ctx, appName, appDescription, appUsage, mockProvider)
		// No panic or error expected
	})

	// Test case: Simulate error setting EnvVarReqLogging
	t.Run("error setting EnvVarReqLogging", func(t *testing.T) {
		// Override the setenv function to simulate an error
		originalSetenv := setenv
		setenv = func(ctx context.Context, key, value string) error {
			if key == gocsi.EnvVarReqLogging {
				return errors.New("mock error")
			}
			return originalSetenv(ctx, key, value)
		}
		defer func() {
			setenv = originalSetenv
		}()

		Run(ctx, appName, appDescription, appUsage, mockProvider)
	})

	// Test case: Simulate error setting EnvVarLogLevel
	t.Run("error setting EnvVarLogLevel", func(t *testing.T) {
		// Override the setenv function to simulate an error
		originalSetenv := setenv
		setenv = func(ctx context.Context, key, value string) error {
			if key == gocsi.EnvVarLogLevel {
				return errors.New("mock error")
			}
			return originalSetenv(ctx, key, value)
		}
		defer func() {
			setenv = originalSetenv
		}()

		Run(ctx, appName, appDescription, appUsage, mockProvider)
	})

	// Test case: Simulate error setting EnvVarRepLogging
	t.Run("error setting EnvVarRepLogging", func(t *testing.T) {
		// Override the setenv function to simulate an error
		originalSetenv := setenv
		setenv = func(ctx context.Context, key, value string) error {
			if key == gocsi.EnvVarRepLogging {
				return errors.New("mock error")
			}
			return originalSetenv(ctx, key, value)
		}
		defer func() {
			setenv = originalSetenv
		}()

		Run(ctx, appName, appDescription, appUsage, mockProvider)
	})
}

func TestPrintUsage(t *testing.T) {
	var appName, appDescription, appUsage, binPath string
	printUsage(appName, appDescription, appUsage, binPath)
}

func TestRmSockFile(t *testing.T) {
	// Mock logrus entry
	log := logrus.New()
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.InfoLevel)

	// Test case: valid listener
	t.Run("valid listener", func(t *testing.T) {
		rmSockFileOnce = sync.Once{}
		listener := &mocks.MockListener{}
		listener.On("Addr").Return(&mocks.MockAddr{NetworkField: "unix", AddressField: "/tmp/mock.sock"})

		rmSockFile(listener)

		// Check if the socket file was removed
		if _, err := os.Stat(listener.Addr().String()); !os.IsNotExist(err) {
			t.Errorf("expected socket file to be removed, but it still exists")
		}
	})

	// Test case: nil listener
	t.Run("nil listener", func(t *testing.T) {
		rmSockFileOnce = sync.Once{}
		rmSockFile(nil)
	})

	// Test case: nil listener address
	t.Run("nil listener address", func(t *testing.T) {
		rmSockFileOnce = sync.Once{}
		listener := &mocks.MockListener{}
		listener.On("Addr").Return(nil)
		rmSockFile(listener)
	})

	// Test case: error removing socket file
	t.Run("error removing socket file", func(t *testing.T) {
		rmSockFileOnce = sync.Once{}

		listener := &mocks.MockListener{}
		listener.On("Addr").Return(&mocks.MockAddr{NetworkField: "unix", AddressField: "/tmp/mock.sock/."})

		rmSockFile(listener)
	})
}

func TestRunMain(t *testing.T) {
	setEnvs(t)

	// Mock the PluginProvider
	mockProvider := new(mocks.MockPluginProvider)
	mockProvider.On("Serve", mock.Anything, mock.Anything).Return(nil)
	mockProvider.On("GracefulStop", mock.Anything).Return()
	mockProvider.On("Stop", mock.Anything).Return()

	runMain(mockProvider)
}
