package main

import (
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
	onExit := func() {
		t.Log("onExit")
	}
	onAbort := func() {
		t.Log("onAbort")
	}
	trapSignals(onExit, onAbort)
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGTERM)
	sigc <- syscall.SIGTERM
	signal.Stop(sigc)
	close(sigc)
}
