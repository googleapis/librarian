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

func TestAPIVersion(t *testing.T) {
	for _, test := range []struct {
		name      string
		overrides *Config
		want      string
	}{
		{
			name:      "No APIs in config",
			overrides: &Config{},
			want:      "",
		},
		{
			name: "API version found",
			overrides: &Config{
				APIs: []API{
					{APIVersion: "v2beta1"},
				},
			},
			want: "v2beta1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := APIVersion(test.overrides)
			if got != test.want {
				t.Errorf("APIVersion() = %v, want %v", got, test.want)
			}
		})
	}
}
