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
	// Name is the fully-qualified name of the service interface
	// (e.g. "google.cloud.speech.v1.Speech").
	Name string `yaml:"name"`

	// Methods contains per-method configuration such as long-running
	// operation polling and batching settings.
	Methods []GAPICMethod `yaml:"methods,omitempty"`
}

// GAPICMethod represents method-level configuration from a gapic yaml file.
type GAPICMethod struct {
	// Name is the simple method name (e.g. "LongRunningRecognize").
	Name string `yaml:"name"`

	// LongRunning contains polling configuration for long-running
	// operations. Nil when the method is not long-running.
	LongRunning *GAPICLongRunning `yaml:"long_running,omitempty"`

	// Batching contains request batching configuration. Nil when the
	// method does not support batching.
	Batching *GAPICBatching `yaml:"batching,omitempty"`
}

// GAPICLongRunning contains polling configuration for long-running operations.
type GAPICLongRunning struct {
	// InitialPollDelayMillis is the delay before the first poll, in
	// milliseconds.
	InitialPollDelayMillis int64 `yaml:"initial_poll_delay_millis,omitempty"`

	// PollDelayMultiplier is the multiplier applied to the poll delay
	// after each attempt.
	PollDelayMultiplier float64 `yaml:"poll_delay_multiplier,omitempty"`

	// MaxPollDelayMillis is the maximum poll delay, in milliseconds.
	MaxPollDelayMillis int64 `yaml:"max_poll_delay_millis,omitempty"`

	// TotalPollTimeoutMillis is the total time allowed for polling
	// before the operation is considered timed out, in milliseconds.
	TotalPollTimeoutMillis int64 `yaml:"total_poll_timeout_millis,omitempty"`
}

// GAPICBatching contains request batching configuration.
type GAPICBatching struct {
	// Thresholds defines the conditions that trigger a batch to be sent.
	Thresholds *GAPICBatchingThresholds `yaml:"thresholds,omitempty"`

	// BatchDescriptor describes how individual requests are combined
	// into a batch.
	BatchDescriptor *GAPICBatchDescriptor `yaml:"batch_descriptor,omitempty"`
}

// GAPICBatchingThresholds defines when batching should occur.
type GAPICBatchingThresholds struct {
	// ElementCountThreshold is the number of elements that triggers a
	// batch to be sent.
	ElementCountThreshold int `yaml:"element_count_threshold,omitempty"`

	// RequestByteThreshold is the total byte size of elements that
	// triggers a batch to be sent.
	RequestByteThreshold int `yaml:"request_byte_threshold,omitempty"`

	// DelayThresholdMillis is the maximum time to wait before sending a
	// batch, in milliseconds.
	DelayThresholdMillis int `yaml:"delay_threshold_millis,omitempty"`

	// FlowControlElementLimit is the maximum number of elements that
	// may be outstanding at once.
	FlowControlElementLimit int `yaml:"flow_control_element_limit,omitempty"`

	// FlowControlByteLimit is the maximum total byte size of elements
	// that may be outstanding at once.
	FlowControlByteLimit int `yaml:"flow_control_byte_limit,omitempty"`

	// FlowControlLimitExceededBehavior controls what happens when the
	// flow control limit is exceeded (e.g. "ThrowException", "Block").
	FlowControlLimitExceededBehavior string `yaml:"flow_control_limit_exceeded_behavior,omitempty"`
}

// GAPICBatchDescriptor describes how requests should be batched together.
type GAPICBatchDescriptor struct {
	// BatchedField is the request field whose values are combined when
	// batching (e.g. "entries" in a logging API).
	BatchedField string `yaml:"batched_field,omitempty"`

	// DiscriminatorFields are request fields that must have identical
	// values for requests to be batched together.
	DiscriminatorFields []string `yaml:"discriminator_fields,omitempty"`

	// SubresponseField is the response field that contains per-element
	// results, used to split a batched response back to individual
	// callers.
	SubresponseField string `yaml:"subresponse_field,omitempty"`
}
