// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

// CommonDiscovery contains discovery-specific configuration for LRO polling.
//
// This is used (via some aliases) by both Swift and Rust, thus the `Common` prefix.
type CommonDiscovery struct {
	// OperationID is the ID of the LRO operation type (e.g., ".google.cloud.compute.v1.Operation").
	OperationID string `yaml:"operation_id"`

	// Pollers is a list of LRO polling configurations.
	Pollers []CommonPoller `yaml:"pollers,omitempty"`
}

// CommonPoller defines how to find a suitable poller RPC for discovery APIs.
//
// This is used (via some aliases) by both Swift and Rust, thus the `Common` prefix.
type CommonPoller struct {
	// Prefix is an acceptable prefix for the URL path (e.g., "compute/v1/projects/{project}/zones/{zone}").
	Prefix string `yaml:"prefix"`

	// MethodID is the corresponding method ID (e.g., ".google.cloud.compute.v1.zoneOperations.get").
	MethodID string `yaml:"method_id"`
}
