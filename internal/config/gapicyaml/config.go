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

// Package gapicyaml provides parsing for GAPIC YAML configuration files.
//
// As of 2025, there are 78 known gapic.yaml files in the googleapis repository:
// https://github.com/search?q=repo%3Agoogleapis%2Fgoogleapis+path%3A*_gapic.yaml&type=code
//
// The <service>_gapic.yaml files are used by the Java GAPIC generator.
package gapicyaml

// Config represents the top-level structure of a GAPIC YAML configuration file.
type Config struct {
	Type                string           `yaml:"type"`
	ConfigSchemaVersion string           `yaml:"config_schema_version"`
	LanguageSettings    LanguageSettings `yaml:"language_settings"`
	Interfaces          []Interface      `yaml:"interfaces,omitempty"`
}

// LanguageSettings contains language-specific configuration settings.
type LanguageSettings struct {
	Java *JavaSettings `yaml:"java,omitempty"`
}

// JavaSettings contains Java-specific configuration.
type JavaSettings struct {
	PackageName    string            `yaml:"package_name,omitempty"`
	InterfaceNames map[string]string `yaml:"interface_names,omitempty"`
}

// Interface represents a service interface configuration.
type Interface struct {
	Name    string   `yaml:"name"`
	Methods []Method `yaml:"methods,omitempty"`
}

// Method represents a method configuration within an interface.
type Method struct {
	Name        string       `yaml:"name"`
	LongRunning *LongRunning `yaml:"long_running,omitempty"`
	Batching    *Batching    `yaml:"batching,omitempty"`
}

// LongRunning contains configuration for long-running operations.
type LongRunning struct {
	InitialPollDelayMillis int64   `yaml:"initial_poll_delay_millis,omitempty"`
	PollDelayMultiplier    float64 `yaml:"poll_delay_multiplier,omitempty"`
	MaxPollDelayMillis     int64   `yaml:"max_poll_delay_millis,omitempty"`
	TotalPollTimeoutMillis int64   `yaml:"total_poll_timeout_millis,omitempty"`
}

// Batching contains configuration for request batching.
type Batching struct {
	Thresholds      *BatchingThresholds `yaml:"thresholds,omitempty"`
	BatchDescriptor *BatchDescriptor    `yaml:"batch_descriptor,omitempty"`
}

// BatchingThresholds defines when batching should occur.
type BatchingThresholds struct {
	ElementCountThreshold            int    `yaml:"element_count_threshold,omitempty"`
	RequestByteThreshold             int    `yaml:"request_byte_threshold,omitempty"`
	DelayThresholdMillis             int    `yaml:"delay_threshold_millis,omitempty"`
	FlowControlElementLimit          int    `yaml:"flow_control_element_limit,omitempty"`
	FlowControlByteLimit             int    `yaml:"flow_control_byte_limit,omitempty"`
	FlowControlLimitExceededBehavior string `yaml:"flow_control_limit_exceeded_behavior,omitempty"`
}

// BatchDescriptor describes how requests should be batched together.
type BatchDescriptor struct {
	BatchedField        string   `yaml:"batched_field,omitempty"`
	DiscriminatorFields []string `yaml:"discriminator_fields,omitempty"`
	SubresponseField    string   `yaml:"subresponse_field,omitempty"`
}
