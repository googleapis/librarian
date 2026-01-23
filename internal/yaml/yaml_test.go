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
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/license"
)

type testConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

func TestUnmarshal(t *testing.T) {
	got, err := Unmarshal[testConfig]([]byte("name: test\nversion: v1.0.0\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := &testConfig{Name: "test", Version: "v1.0.0"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestUnmarshalError(t *testing.T) {
	_, err := Unmarshal[testConfig]([]byte("name: [invalid"))
	if err == nil {
		t.Error("Unmarshal() expected error for invalid YAML")
	}
}

func TestMarshal(t *testing.T) {
	input := &testConfig{Name: "test", Version: "v1.0.0"}
	data, err := Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Unmarshal[testConfig](data)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(input, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestReadWrite(t *testing.T) {
	want := &testConfig{Name: "test", Version: "v1.0.0"}
	path := filepath.Join(t.TempDir(), "test.yaml")
	if err := Write(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Read[testConfig](path)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestWrite(t *testing.T) {
	// Construct expected header
	lines := license.LicenseHeader("2026")
	var header string
	for _, line := range lines {
		if line == "" {
			header += "#\n"
		} else {
			header += "#" + line + "\n"
		}
	}
	header += "\n"

	want := header + `name: test
version: v1.0.0
`
	path := filepath.Join(t.TempDir(), "test.yaml")
	if err := Write(path, &testConfig{Name: "test", Version: "v1.0.0"}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, string(got)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestReadError(t *testing.T) {
	_, err := Read[testConfig]("/nonexistent/path/file.yaml")
	if err == nil {
		t.Error("Read() expected error for nonexistent file")
	}
}

func TestWriteError(t *testing.T) {
	err := Write("/nonexistent/path/file.yaml", &testConfig{Name: "test"})
	if err == nil {
		t.Error("Write() expected error for invalid path")
	}
}

func TestStringSlice_EmptySlice(t *testing.T) {
	strSlice := StringSlice{}
	got := strSlice.IsZero()
	if diff := cmp.Diff(false, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestStringSlice_NilSlice(t *testing.T) {
	var strSlice StringSlice
	got := strSlice.IsZero()
	if diff := cmp.Diff(true, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
