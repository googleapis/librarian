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

package gcloud

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"cloud.google.com/go/librarian/internal/yaml"
	"github.com/google/go-cmp/cmp"
)

func TestReadGcloudConfig(t *testing.T) {
	cfg, err := yaml.Read[Config]("testdata/parallelstore/gcloud.yaml")
	if err != nil {
		t.Fatal(err)
	}

	got, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal struct to YAML: %v", err)
	}

	var index int
	data, err := os.ReadFile("testdata/parallelstore/gcloud.yaml")
	if err != nil {
		t.Fatalf("failed to read temporary YAML file: %v", err)
	}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "#") {
			// Skip the header, and the new lines after the header
			index = i + 2
			continue
		}
	}
	want := fmt.Sprintf("service_name: %s\n%s", cfg.ServiceName, strings.Join(lines[index:], "\n"))
	if diff := cmp.Diff(want, string(got)); diff != "" {
		t.Errorf("mismatch(-want, +got)\n%s", diff)
	}
}
