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

package provider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestReadGcloudConfig(t *testing.T) {
	const validConfig = `
service_name: "example.googleapis.com"
generate_operations: true
apis:
- name: "ExampleAPI"
  api_version: "v1"
  supports_star_update_masks: true
  root_is_hidden: true
  release_tracks: ["GA", "BETA"]
  help_text:
    service_rules:
    - selector: "example.v1.Service"
      help_text:
        brief: "Brief help text"
        description: "Detailed help text"
        examples: ["example command"]
  output_formatting:
  - selector: "example.v1.Service.Method"
    format: "table(name)"
  command_operations_config:
  - selector: "example.v1.Service.Method"
    display_operation_result: true
resource_patterns:
- type: "example.googleapis.com/Resource"
  patterns: ["projects/{project}/resources/{resource}"]
  api_version: "v1"
`

	tmpFile := filepath.Join(t.TempDir(), "gcloud.yaml")
	if err := os.WriteFile(tmpFile, []byte(validConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := ReadGcloudConfig(tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	marshaled, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}

	roundTripped, err := yaml.Unmarshal[Config](marshaled)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(cfg, roundTripped); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestReadGcloudConfig_Invalid(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name: "missing resource pattern type",
			content: `service_name: "foo"
resource_patterns: [{patterns: ["a/b"]}]`,
			wantErr: "resource_patterns[0].type is required",
		},
		{
			name: "empty resource patterns list",
			content: `service_name: "foo"
resource_patterns: [{type: "foo/Bar", patterns: []}]`,
			wantErr: "resource_patterns[0].patterns must not be empty",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpFile := filepath.Join(t.TempDir(), "gcloud.yaml")
			if err := os.WriteFile(tmpFile, []byte(tc.content), 0644); err != nil {
				t.Fatal(err)
			}

			_, err := ReadGcloudConfig(tmpFile)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}
