// Copyright 2026 Google LLC
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

package serviceconfig

// GAPICYamlConfig contains all configuration imported from a *_gapic.yaml
// file. This groups Java-specific and language-neutral settings so they can
// be written together to a single file during generation.
type GAPICYamlConfig struct {
	// JavaPackageName is the Java package name.
	// TODO(https://github.com/googleapis/librarian/issues/3041): move to librarian.yaml.
	JavaPackageName string `yaml:"java_package_name,omitempty"`

	// JavaInterfaceNames maps fully-qualified interface names to short
	// Java class names.
	// TODO(https://github.com/googleapis/librarian/issues/3041): move to librarian.yaml.
	JavaInterfaceNames map[string]string `yaml:"java_interface_names,omitempty"`

	// Interfaces contains language-neutral per-interface, per-method
	// configuration including long-running operation polling settings and
	// batching config.
	Interfaces []GAPICInterface `yaml:"interfaces,omitempty"`
}

// GAPICInterface represents a service interface with method-level config
// imported from *_gapic.yaml files in googleapis.
type GAPICInterface struct {
	Name    string        `yaml:"name"`
	Methods []GAPICMethod `yaml:"methods,omitempty"`
}

// GAPICMethod represents method-level configuration from a gapic yaml file.
type GAPICMethod struct {
	Name        string            `yaml:"name"`
	LongRunning *GAPICLongRunning `yaml:"long_running,omitempty"`
	Batching    *GAPICBatching    `yaml:"batching,omitempty"`
}

// GAPICLongRunning contains polling configuration for long-running operations.
type GAPICLongRunning struct {
	InitialPollDelayMillis int64   `yaml:"initial_poll_delay_millis,omitempty"`
	PollDelayMultiplier    float64 `yaml:"poll_delay_multiplier,omitempty"`
	MaxPollDelayMillis     int64   `yaml:"max_poll_delay_millis,omitempty"`
	TotalPollTimeoutMillis int64   `yaml:"total_poll_timeout_millis,omitempty"`
}

// GAPICBatching contains request batching configuration.
type GAPICBatching struct {
	Thresholds      *GAPICBatchingThresholds `yaml:"thresholds,omitempty"`
	BatchDescriptor *GAPICBatchDescriptor    `yaml:"batch_descriptor,omitempty"`
}

// GAPICBatchingThresholds defines when batching should occur.
type GAPICBatchingThresholds struct {
	ElementCountThreshold            int    `yaml:"element_count_threshold,omitempty"`
	RequestByteThreshold             int    `yaml:"request_byte_threshold,omitempty"`
	DelayThresholdMillis             int    `yaml:"delay_threshold_millis,omitempty"`
	FlowControlElementLimit          int    `yaml:"flow_control_element_limit,omitempty"`
	FlowControlByteLimit             int    `yaml:"flow_control_byte_limit,omitempty"`
	FlowControlLimitExceededBehavior string `yaml:"flow_control_limit_exceeded_behavior,omitempty"`
}

// GAPICBatchDescriptor describes how requests should be batched together.
type GAPICBatchDescriptor struct {
	BatchedField        string   `yaml:"batched_field,omitempty"`
	DiscriminatorFields []string `yaml:"discriminator_fields,omitempty"`
	SubresponseField    string   `yaml:"subresponse_field,omitempty"`
}
