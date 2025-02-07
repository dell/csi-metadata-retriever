package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"testing"
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
	// Test case: SIGTERM
	onExitCalled := false
	onExit := func() {
		onExitCalled = true
	}
	onAbort := func() {}
	trapSignals(onExit, onAbort)
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGTERM)
	sigc <- syscall.SIGTERM
	signal.Stop(sigc)
	close(sigc)
	if onExitCalled {
		t.Error("Expected onExit to be called")
	}

	// Test case: SIGHUP
	onExitCalled = false
	onAbortCalled := false
	trapSignals(onExit, onAbort)
	sigc = make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGHUP)
	sigc <- syscall.SIGHUP
	signal.Stop(sigc)
	close(sigc)
	if onExitCalled {
		t.Error("Expected onExit to be called")
	}
	if onAbortCalled {
		t.Error("Expected onAbort to not be called")
	}

	// Test case: SIGINT
	onExitCalled = false
	onAbortCalled = false
	trapSignals(onExit, onAbort)
	sigc = make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT)
	sigc <- syscall.SIGINT
	signal.Stop(sigc)
	close(sigc)
	if onExitCalled {
		t.Error("Expected onExit to be called")
	}
	if onAbortCalled {
		t.Error("Expected onAbort to not be called")
	}

	// Test case: SIGQUIT
	onExitCalled = false
	onAbortCalled = false
	trapSignals(onExit, onAbort)
	sigc = make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGQUIT)
	sigc <- syscall.SIGQUIT
	signal.Stop(sigc)
	close(sigc)
	if onExitCalled {
		t.Error("Expected onExit to be called")
	}
	if onAbortCalled {
		t.Error("Expected onAbort to not be called")
	}

	// Test case: SIGABRT
	onExitCalled = false
	onAbortCalled = false
	trapSignals(onExit, onAbort)
	sigc = make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGABRT)
	sigc <- syscall.SIGABRT
	signal.Stop(sigc)
	close(sigc)
	if onExitCalled {
		t.Error("Expected onExit to not be called")
	}
	if onAbortCalled {
		t.Error("Expected onAbort to be called")
	}
}

type MockPluginProvider struct {
}

func (m *MockPluginProvider) Serve(ctx context.Context, l net.Listener) error {
	return nil
}

func (m *MockPluginProvider) GracefulStop(ctx context.Context) {
}

func (m *MockPluginProvider) Stop(ctx context.Context) {
}

func TestRun(t *testing.T) {
	var appName, appDescription, appUsage string
	ctx := context.Background()
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

	Run(ctx, appName, appDescription, appUsage, &MockPluginProvider{})
}

func TestPrintUsage(t *testing.T) {
	var appName, appDescription, appUsage, binPath string
	printUsage(appName, appDescription, appUsage, binPath)
}

func TestRmSockFile(t *testing.T) {
	var l net.Listener
	rmSockFile(l)
}
