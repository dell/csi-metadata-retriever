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

package service

const (
	// Name is the name of this CSI SP.
	Name = "csi-metadata-retriever"

	// VendorVersion is the version of this CSP SP.
	VendorVersion = "1.0.0"
)

// Service is a CSI SP and idempotency.Provider.
type Service interface {
	//	retriever.MetadataRetrieverClient
}

type service struct{}

// New returns a new Service.
func New() Service {
	return &service{}
}
