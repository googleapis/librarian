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

package rust

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	sidekickconfig "github.com/googleapis/librarian/internal/sidekick/config"
)

func TestToSidekickReleaseConfig(t *testing.T) {
	tests := []struct {
		name  string
		input *config.Release
		want  *sidekickconfig.Release
	}{
		{
			name:  "nil input",
			input: nil,
			want:  nil,
		},
		{
			name: "basic config",
			input: &config.Release{
				Remote: "test-remote",
				Branch: "test-branch",
				Tools: map[string][]config.Tool{
					"go": {
						{
							Name:    "goreleaser",
							Version: "1.0.0",
						},
					},
				},
			},
			want: &sidekickconfig.Release{
				Remote: "test-remote",
				Branch: "test-branch",
				Tools: map[string][]sidekickconfig.Tool{
					"go": {
						{
							Name:    "goreleaser",
							Version: "1.0.0",
						},
					},
				},
			},
		},
		{
			name: "full config",
			input: &config.Release{
				Remote: "full-remote",
				Branch: "full-branch",
				Tools: map[string][]config.Tool{
					"go": {
						{
							Name:    "goreleaser",
							Version: "1.2.3",
						},
					},
					"java": {
						{
							Name:    "maven",
							Version: "3.8.1",
						},
						{
							Name:    "gradle",
							Version: "7.4.0",
						},
					},
				},
				Preinstalled: map[string]string{
					"tool1": "/path/to/tool1",
					"tool2": "/path/to/tool2",
				},
				IgnoredChanges: []string{"file1.go", "file2.java"},
				RootsPem:       "-----BEGIN CERTIFICATE----...",
			},
			want: &sidekickconfig.Release{
				Remote: "full-remote",
				Branch: "full-branch",
				Tools: map[string][]sidekickconfig.Tool{
					"go": {
						{
							Name:    "goreleaser",
							Version: "1.2.3",
						},
					},
					"java": {
						{
							Name:    "maven",
							Version: "3.8.1",
						},
						{
							Name:    "gradle",
							Version: "7.4.0",
						},
					},
				},
				Preinstalled: map[string]string{
					"tool1": "/path/to/tool1",
					"tool2": "/path/to/tool2",
				},
				IgnoredChanges: []string{"file1.go", "file2.java"},
				RootsPem:       "-----BEGIN CERTIFICATE----...",
			},
		},
	}

	for _, test := range tests {
		test := test // Capture range variable
		t.Run(test.name, func(t *testing.T) {
			got := ToSidekickReleaseConfig(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("toSidekickReleaseConfig() mismatch (-want +got):%s", diff)
			}
		})
	}
}

func TestToConfigTools(t *testing.T) {
	tests := []struct {
		name        string
		input       []sidekickconfig.Tool
		expected    []config.Tool
		expectedNil bool
	}{
		{
			name:        "nil input",
			input:       nil,
			expected:    nil,
			expectedNil: true,
		},
		{
			name:        "empty slice",
			input:       []sidekickconfig.Tool{},
			expected:    []config.Tool{},
			expectedNil: false,
		},
		{
			name: "valid tools",
			input: []sidekickconfig.Tool{
				{Name: "tool1", Version: "1.0.0"},
				{Name: "tool2", Version: "2.0.0"},
			},
			expected: []config.Tool{
				{Name: "tool1", Version: "1.0.0"},
				{Name: "tool2", Version: "2.0.0"},
			},
			expectedNil: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ToConfigTools(test.input)
			if test.expectedNil && got != nil {
				t.Errorf("ToConfigTools() got = %v, want nil", got)
			}
			if !test.expectedNil && len(got) != len(test.expected) {
				t.Errorf("ToConfigTools() got = %v, want %v", got, test.expected)
			}
			for i, v := range got {
				if v.Name != test.expected[i].Name || v.Version != test.expected[i].Version {
					t.Errorf("ToConfigTools() got = %v, want %v", got, test.expected)
					break
				}
			}
		})
	}
}
