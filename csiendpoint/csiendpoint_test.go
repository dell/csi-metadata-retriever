/*
 *
 * Copyright © 2021-2024 Dell Inc. or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

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

package csiendpoint

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test function outside the csiendpoint package that makes use of the mock.
// TestGetCSIEndpoint tests the GetCSIEndpoint function.
func TestGetCSIEndpoint(t *testing.T) {
	tests := []struct {
		name             string
		csiEndpointEnv   string
		expectedNetwork  string
		expectedAddr     string
		expectedErrorMsg string
	}{
		{
			name:             "EnvVar Not Set",
			csiEndpointEnv:   "",
			expectedNetwork:  "",
			expectedAddr:     "",
			expectedErrorMsg: "missing CSI_RETRIEVER_ENDPOINT",
		},
		{
			name:             "Valid EnvVar",
			csiEndpointEnv:   "tcp://127.0.0.1:10000",
			expectedNetwork:  "tcp",
			expectedAddr:     "127.0.0.1:10000",
			expectedErrorMsg: "",
		},
		{
			name:             "Invalid EnvVar",
			csiEndpointEnv:   "invalid",
			expectedNetwork:  "unix",
			expectedAddr:     "invalid",
			expectedErrorMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("CSI_RETRIEVER_ENDPOINT", tt.csiEndpointEnv)
			defer os.Unsetenv("CSI_RETRIEVER_ENDPOINT")

			network, addr, err := GetCSIEndpoint()
			if tt.expectedErrorMsg == "" {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedNetwork, network)
				assert.Equal(t, tt.expectedAddr, addr)
			} else {
				assert.EqualError(t, err, tt.expectedErrorMsg)
				assert.Empty(t, network)
				assert.Empty(t, addr)
			}
		})
	}
}

// TestGetCSIEndpointListener tests the GetCSIEndpointListener function.
func TestGetCSIEndpointListener(t *testing.T) {
	tests := []struct {
		name            string
		csiEndpointEnv  string
		expectedNetwork string
		expectedAddr    string
		expectedError   string
	}{
		{
			name:           "EnvVar Not Set",
			csiEndpointEnv: "",
			expectedError:  "missing CSI_RETRIEVER_ENDPOINT",
		},
		{
			name:            "Valid EnvVar",
			csiEndpointEnv:  "tcp://127.0.0.1:10000",
			expectedNetwork: "tcp",
			expectedAddr:    "127.0.0.1:10000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("CSI_RETRIEVER_ENDPOINT", tt.csiEndpointEnv)
			defer os.Unsetenv("CSI_RETRIEVER_ENDPOINT")

			listener, err := GetCSIEndpointListener()
			if tt.expectedError == "" {
				require.NoError(t, err)
				assert.NotNil(t, listener)
				assert.Equal(t, tt.expectedNetwork, listener.Addr().Network())
				assert.Equal(t, tt.expectedAddr, listener.Addr().String())
			} else {
				assert.EqualError(t, err, tt.expectedError)
				assert.Nil(t, listener)
			}
		})
	}
}
