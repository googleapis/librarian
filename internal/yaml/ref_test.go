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

package yaml

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type refConfig struct {
	Ref RefString `yaml:"ref"`
}

func TestRefStringMarshal(t *testing.T) {
	input := &refConfig{Ref: RefString("some/path")}
	data, err := yaml.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "!REF") {
		t.Errorf("Marshal() output missing !REF tag: %s", string(data))
	}
}

func TestRefStringUnmarshal(t *testing.T) {
	data := "ref: !REF some/path\n"
	var got refConfig
	if err := yaml.Unmarshal([]byte(data), &got); err != nil {
		t.Fatal(err)
	}
	if got.Ref != "some/path" {
		t.Errorf("Unmarshal() ref = %q, want %q", got.Ref, "some/path")
	}
}

func TestRefStringUnmarshalError(t *testing.T) {
	data := "ref: some/path\n"
	var got refConfig
	err := yaml.Unmarshal([]byte(data), &got)
	if err == nil {
		t.Error("Unmarshal() expected error for missing !REF tag")
	}
}

func TestRefStringRoundTrip(t *testing.T) {
	input := &refConfig{Ref: RefString("some/path")}
	data, err := yaml.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	var got refConfig
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Ref != input.Ref {
		t.Errorf("round-trip ref = %q, want %q", got.Ref, input.Ref)
	}
}
