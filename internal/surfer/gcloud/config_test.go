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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestReadGcloudConfig(t *testing.T) {
	cfg, err := yaml.Read[Config]("testdata/parallelstore/gcloud.yaml")
	if err != nil {
		t.Fatal(err)
	}

	// Round-trip through marshal/unmarshal and compare structs directly.
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}

	roundTripped, err := yaml.Unmarshal[Config](data)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(cfg, roundTripped, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
