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

	// Mock exit function - save and restore original
	originalExit := exit
	exit = func(_ int) {}
	defer func() {
		exit = originalExit
	}()

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
			trapSignals(onExit)

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
	os.Setenv("X_CSI_LOG_LEVEL", "debug")
	os.Setenv("X_CSI_PLUGIN_INFO", "my-plugin")
	os.Setenv("X_CSI_REQ_ID_INJECTION", "true")
	os.Setenv("X_CSI_SPEC_VALIDATION", "true")
	os.Setenv("X_CSI_SPEC_REQ_VALIDATION", "true")
	os.Setenv("X_CSI_SPEC_REP_VALIDATION", "true")
	os.Setenv("X_CSI_SPEC_DISABLE_LEN_CHECK", "true")
}

func TestRun(t *testing.T) {
	var appName, appDescription, appUsage string
	ctx := context.Background()
	setEnvs(t)

	// Mock exit function for all subtests
	originalExit := exit
	exit = func(_ int) {}
	defer func() {
		exit = originalExit
	}()

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
	t.Run("help flag", func(_ *testing.T) {
		os.Args = []string{"cmd", "-?"}
		Run(ctx, appName, appDescription, appUsage, mockProvider)
		// No panic or error expected
	})

	// Test case: no endpoint set
	t.Run("no endpoint set", func(_ *testing.T) {
		os.Unsetenv("CSI_RETRIEVER_ENDPOINT")
		Run(ctx, appName, appDescription, appUsage, mockProvider)
		// No panic or error expected
	})

	// Test case: multiple runs with different endpoints
	t.Run("multiple runs", func(_ *testing.T) {
		setEnvs(t)
		Run(ctx, appName, appDescription, appUsage, mockProvider)
		Run(ctx, appName, appDescription, appUsage, mockProvider)
	})

	// Test case: listener error
	t.Run("listener error", func(t *testing.T) {
		setEnvs(t)
		var mu sync.Mutex
		originalExit := exit
		exitCalled := false
		exit = func(_ int) {
			mu.Lock()
			exitCalled = true
			mu.Unlock()
		}
		defer func() {
			exit = originalExit
		}()
		originalListener := getCSIEndpointListener
		getCSIEndpointListener = func() (net.Listener, error) {
			return nil, errors.New("listener error")
		}
		defer func() {
			getCSIEndpointListener = originalListener
		}()
		Run(ctx, appName, appDescription, appUsage, mockProvider)
		mu.Lock()
		if !exitCalled {
			t.Errorf("expected exit to be called on listener error")
		}
		mu.Unlock()
	})

	// Test case: serve error
	t.Run("serve error", func(t *testing.T) {
		setEnvs(t)
		var mu sync.Mutex
		originalExit := exit
		exitCalled := false
		exit = func(_ int) {
			mu.Lock()
			exitCalled = true
			mu.Unlock()
		}
		defer func() {
			exit = originalExit
		}()
		mockProvider2 := new(mocks.MockPluginProvider)
		mockProvider2.On("Serve", mock.Anything, mock.Anything).Return(errors.New("serve error"))
		mockProvider2.On("GracefulStop", mock.Anything).Return()
		mockProvider2.On("Stop", mock.Anything).Return()
		mockListener := &mocks.MockListener{}
		mockListener.On("Addr").Return(&mocks.MockAddr{NetworkField: "unix", AddressField: "/tmp/test.sock"})
		rmSockFileOnce = sync.Once{}
		getCSIEndpointListener = func() (net.Listener, error) {
			return mockListener, nil
		}
		Run(ctx, appName, appDescription, appUsage, mockProvider2)
		mu.Lock()
		if !exitCalled {
			t.Errorf("expected exit to be called on serve error")
		}
		mu.Unlock()
	})
}

func TestTrapSignalsCallback(t *testing.T) {
	callbackExecuted := false
	onExit := func() {
		callbackExecuted = true
	}

	// Start trap signals
	trapSignals(onExit)

	// Give goroutine time to start
	time.Sleep(100 * time.Millisecond)

	// Callback should not be called without a signal
	if callbackExecuted {
		t.Errorf("callback should not be executed without signal")
	}
}

func TestPrintUsage(_ *testing.T) {
	appName := "TestApp"
	appDescription := "Test Description"
	appUsage := "test usage"
	binPath := "test-bin"
	printUsage(appName, appDescription, appUsage, binPath)

	// Test with empty values
	printUsage("", "", "", "")
}

func TestRmSockFile(t *testing.T) {
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
	t.Run("nil listener", func(_ *testing.T) {
		rmSockFileOnce = sync.Once{}
		rmSockFile(nil)
	})

	// Test case: nil listener address
	t.Run("nil listener address", func(_ *testing.T) {
		rmSockFileOnce = sync.Once{}
		listener := &mocks.MockListener{}
		listener.On("Addr").Return(nil)
		rmSockFile(listener)
	})

	// Test case: error removing socket file
	t.Run("error removing socket file", func(_ *testing.T) {
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
