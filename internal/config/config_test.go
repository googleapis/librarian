// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

func TestReadWrite(t *testing.T) {
	cfg, err := Read("testdata/rust/librarian.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var got bytes.Buffer
	enc := yaml.NewEncoder(&got)
	enc.SetIndent(2)
	if err := enc.Encode(cfg); err != nil {
		t.Fatal(err)
	}

	wantCfg, err := Read("testdata/rust/librarian.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var gotCfg Config
	if err := yaml.Unmarshal(got.Bytes(), &gotCfg); err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(wantCfg, &gotCfg); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
