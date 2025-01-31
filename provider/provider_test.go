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

package provider

import (
	"context"
	"net"
	"testing"

	"github.com/dell/csi-metadata-retriever/retriever"
	"github.com/stretchr/testify/assert"
)

// Mock net.Listener implementation for testing.
type mockListener struct{}

func (m *mockListener) Accept() (net.Conn, error) {
	return nil, nil
}
func (m *mockListener) Close() error {
	return nil
}
func (m *mockListener) Addr() net.Addr {
	return nil
}

func TestNew(t *testing.T) {
	tests := []struct {
		name                string
		expectedEnvVars     []string
		expectedBeforeServe func(context.Context, *retriever.Plugin, net.Listener) error
	}{
		{
			name: "New PluginProvider",
			expectedEnvVars: []string{
				"X_CSI_SPEC_REQ_VALIDATION=true",
				"X_CSI_SERIAL_VOL_ACCESS=true",
			},
			expectedBeforeServe: func(ctx context.Context, plugin *retriever.Plugin, listener net.Listener) error {
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pp := New()
			plugin, ok := pp.(*retriever.Plugin)
			assert.True(t, ok)

			assert.ElementsMatch(t, tt.expectedEnvVars, plugin.EnvVars)

			// Testing BeforeServe
			err := plugin.BeforeServe(context.Background(), plugin, &mockListener{})
			assert.NoError(t, err)
		})
	}
}
