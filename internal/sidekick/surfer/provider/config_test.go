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

package provider

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFindHelpTextRule(t *testing.T) {
	for _, test := range []struct {
		name      string
		overrides *Config
		methodID  string
		want      *HelpTextRule
	}{
		{
			name:      "No APIs in config",
			overrides: &Config{},
			methodID:  "google.cloud.test.v1.Service.CreateInstance",
			want:      nil,
		},
		{
			name: "Matching rule found",
			overrides: &Config{
				APIs: []API{
					{
						HelpText: &HelpTextRules{
							MethodRules: []*HelpTextRule{
								{
									Selector: "google.cloud.test.v1.Service.CreateInstance",
									HelpText: &HelpTextElement{
										Brief: "Override Brief",
									},
								},
							},
						},
					},
				},
			},
			methodID: "google.cloud.test.v1.Service.CreateInstance",
			want: &HelpTextRule{
				Selector: "google.cloud.test.v1.Service.CreateInstance",
				HelpText: &HelpTextElement{
					Brief: "Override Brief",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := FindHelpTextRule(test.overrides, test.methodID)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("FindHelpTextRule() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindFieldHelpTextRule(t *testing.T) {
	for _, test := range []struct {
		name      string
		overrides *Config
		fieldID   string
		want      *HelpTextRule
	}{
		{
			name:      "No APIs in config",
			overrides: &Config{},
			fieldID:   ".google.cloud.test.v1.Request.instance_id",
			want:      nil,
		},
		{
			name: "Matching rule found",
			overrides: &Config{
				APIs: []API{
					{
						HelpText: &HelpTextRules{
							FieldRules: []*HelpTextRule{
								{
									Selector: ".google.cloud.test.v1.Request.instance_id",
									HelpText: &HelpTextElement{
										Brief: "Override Field Brief",
									},
								},
							},
						},
					},
				},
			},
			fieldID: ".google.cloud.test.v1.Request.instance_id",
			want: &HelpTextRule{
				Selector: ".google.cloud.test.v1.Request.instance_id",
				HelpText: &HelpTextElement{
					Brief: "Override Field Brief",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := FindFieldHelpTextRule(test.overrides, test.fieldID)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("FindFieldHelpTextRule() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAPI(t *testing.T) {
	api1 := API{Name: "parallelstore.googleapis.com", APIVersion: "v1"}
	api2 := API{Name: "parallelstore.googleapis.com", APIVersion: "v1beta1"}
	api3 := API{Name: "cloudbuild.googleapis.com", APIVersion: "v1"}

	for _, test := range []struct {
		name        string
		config      *Config
		serviceName string
		version     string
		want        *API
	}{
		{
			name:        "Nil Config",
			config:      nil,
			serviceName: "parallelstore.googleapis.com",
			version:     "v1",
			want:        nil,
		},
		{
			name:        "No APIs in Config",
			config:      &Config{},
			serviceName: "parallelstore.googleapis.com",
			version:     "v1",
			want:        nil,
		},
		{
			name: "Exact match",
			config: &Config{
				APIs: []API{api1, api2, api3},
			},
			serviceName: "parallelstore.googleapis.com",
			version:     "v1beta1",
			want:        &api2,
		},
		{
			name: "No fallback to single API in config",
			config: &Config{
				APIs: []API{api3},
			},
			serviceName: "parallelstore.googleapis.com",
			version:     "v1",
			want:        nil,
		},
		{
			name: "No match with multiple APIs",
			config: &Config{
				APIs: []API{api1, api2, api3},
			},
			serviceName: "nonexistent.googleapis.com",
			version:     "v2",
			want:        nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := test.config.API(test.serviceName, test.version)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("API() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSupportsStarUpdateMasks(t *testing.T) {
	tBool := true
	fBool := false

	for _, test := range []struct {
		name        string
		config      *Config
		serviceName string
		version     string
		want        bool
	}{
		{
			name:        "Nil Config",
			config:      nil,
			serviceName: "parallelstore.googleapis.com",
			version:     "v1",
			want:        true,
		},
		{
			name:        "No APIs in Config",
			config:      &Config{},
			serviceName: "parallelstore.googleapis.com",
			version:     "v1",
			want:        true,
		},
		{
			name: "SupportsStarUpdateMasks omitted (default true)",
			config: &Config{
				APIs: []API{
					{Name: "parallelstore.googleapis.com", APIVersion: "v1"},
				},
			},
			serviceName: "parallelstore.googleapis.com",
			version:     "v1",
			want:        true,
		},
		{
			name: "Explicitly set to true",
			config: &Config{
				APIs: []API{
					{Name: "parallelstore.googleapis.com", APIVersion: "v1", SupportsStarUpdateMasks: &tBool},
				},
			},
			serviceName: "parallelstore.googleapis.com",
			version:     "v1",
			want:        true,
		},
		{
			name: "Explicitly set to false",
			config: &Config{
				APIs: []API{
					{Name: "parallelstore.googleapis.com", APIVersion: "v1", SupportsStarUpdateMasks: &fBool},
				},
			},
			serviceName: "parallelstore.googleapis.com",
			version:     "v1",
			want:        false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := SupportsStarUpdateMasks(test.config, test.serviceName, test.version)
			if got != test.want {
				t.Errorf("SupportsStarUpdateMasks() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestOutputFormat(t *testing.T) {
	for _, test := range []struct {
		name      string
		overrides *Config
		methodID  string
		want      string
	}{
		{
			name:      "Nil Config",
			overrides: nil,
			methodID:  "google.cloud.test.v1.Service.ListInstances",
			want:      "",
		},
		{
			name:      "No APIs in Config",
			overrides: &Config{},
			methodID:  "google.cloud.test.v1.Service.ListInstances",
			want:      "",
		},
		{
			name: "Matching selector found (exact selector)",
			overrides: &Config{
				APIs: []API{
					{
						OutputFormatting: []*OutputFormatting{
							{
								Selector: "google.cloud.test.v1.Service.ListInstances",
								Format:   "table(name, createTime)",
							},
						},
					},
				},
			},
			methodID: "google.cloud.test.v1.Service.ListInstances",
			want:     "table(name, createTime)",
		},
		{
			name: "Matching selector with leading dot in methodID",
			overrides: &Config{
				APIs: []API{
					{
						OutputFormatting: []*OutputFormatting{
							{
								Selector: "google.cloud.test.v1.Service.ListInstances",
								Format:   "table(name, createTime)",
							},
						},
					},
				},
			},
			methodID: ".google.cloud.test.v1.Service.ListInstances",
			want:     "table(name, createTime)",
		},
		{
			name: "No matching selector",
			overrides: &Config{
				APIs: []API{
					{
						OutputFormatting: []*OutputFormatting{
							{
								Selector: "google.cloud.test.v1.Service.OtherMethod",
								Format:   "table(name)",
							},
						},
					},
				},
			},
			methodID: "google.cloud.test.v1.Service.ListInstances",
			want:     "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := OutputFormat(test.overrides, test.methodID)
			if got != test.want {
				t.Errorf("OutputFormat() = %v, want %v", got, test.want)
			}
		})
	}
}
