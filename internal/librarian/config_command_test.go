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

package librarian

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/yaml"
)

// TestConfigCommand_Get tests that the config get command successfully reads a value from librarian.yaml.
func TestConfigCommand_Get(t *testing.T) {
	for _, test := range []struct {
		name       string
		path       string
		configYAML string
		want       string
	}{
		{
			name:       "get string value",
			path:       "version",
			configYAML: "version: 1.2.3\n",
			want:       "1.2.3\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			t.Chdir(tempDir)

			if err := yaml.Write("librarian.yaml", []byte(test.configYAML), 0644); err != nil {
				t.Fatal(err)
			}

			var buf bytes.Buffer
			err := runConfigGet(&buf, test.path)

			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.want, buf.String()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestConfigCommand_Get_Error tests that the config get command returns an error when the path is missing or the key is not found.
func TestConfigCommand_Get_Error(t *testing.T) {
	for _, test := range []struct {
		name       string
		path       string
		configYAML string
		wantErr    error
	}{
		{
			name:       "missing path",
			path:       "",
			configYAML: "version: 1.2.3\n",
			wantErr:    errPathRequired,
		},
		{
			name:       "key not found",
			path:       "foo",
			configYAML: "version: 1.2.3\n",
			wantErr:    errUnsupportedPath,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			t.Chdir(tempDir)

			if err := os.WriteFile("librarian.yaml", []byte(test.configYAML), 0644); err != nil {
				t.Fatal(err)
			}

			var buf bytes.Buffer
			err := runConfigGet(&buf, test.path)

			if err == nil {
				t.Fatal("expected error; got nil")
			}

			if !errors.Is(err, test.wantErr) {
				t.Fatalf("got error %v, want %v", err, test.wantErr)
			}
		})
	}
}

// TestConfigCommand_Set tests that the config set command successfully updates a value in librarian.yaml.
func TestConfigCommand_Set(t *testing.T) {
	for _, test := range []struct {
		name       string
		path       string
		value      string
		configYAML string
		wantYAML   string
	}{
		{
			name:       "set string value",
			path:       "version",
			value:      "1.2.4",
			configYAML: "version: 1.2.3\n",
			wantYAML:   "version: 1.2.4\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			t.Chdir(tempDir)

			if err := os.WriteFile("librarian.yaml", []byte(test.configYAML), 0644); err != nil {
				t.Fatal(err)
			}

			err := runConfigSet(test.path, test.value)

			if err != nil {
				t.Fatal(err)
			}

			gotYAML, err := os.ReadFile("librarian.yaml")
			if err != nil {
				t.Fatal(err)
			}

			if !strings.HasSuffix(string(gotYAML), test.wantYAML) {
				t.Errorf("got YAML =\n%s\nwant ending with %q", string(gotYAML), test.wantYAML)
			}
		})
	}
}

// TestConfigCommand_Set_Error tests that the config set command returns an error when the path or value is missing.
func TestConfigCommand_Set_Error(t *testing.T) {
	for _, test := range []struct {
		name       string
		path       string
		value      string
		configYAML string
		wantErr    error
	}{
		{
			name:       "missing path",
			path:       "",
			value:      "",
			configYAML: "version: 1.2.3\n",
			wantErr:    errPathRequired,
		},
		{
			name:       "missing value",
			path:       "version",
			value:      "",
			configYAML: "version: 1.2.3\n",
			wantErr:    errValueRequired,
		},
		{
			name:       "unsupported path",
			path:       "foo",
			value:      "bar",
			configYAML: "version: 1.2.3\n",
			wantErr:    errUnsupportedPath,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			t.Chdir(tempDir)

			if err := os.WriteFile("librarian.yaml", []byte(test.configYAML), 0644); err != nil {
				t.Fatal(err)
			}

			err := runConfigSet(test.path, test.value)

			if err == nil {
				t.Fatal("expected error; got nil")
			}

			if !errors.Is(err, test.wantErr) {
				t.Fatalf("got error %v, want %v", err, test.wantErr)
			}
		})
	}
}

// TestConfigCommand_Set_ReadError tests that the config set command returns an error when librarian.yaml is unreadable.
func TestConfigCommand_Set_ReadError(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	if err := os.WriteFile("librarian.yaml", []byte("version: 1.2.3\n"), 0000); err != nil {
		t.Fatal(err)
	}

	err := runConfigSet("version", "1.2.4")

	if err == nil {
		t.Fatal("expected error; got nil")
	}

	if !errors.Is(err, os.ErrPermission) {
		t.Fatalf("got error %v, want %v", err, os.ErrPermission)
	}
}
