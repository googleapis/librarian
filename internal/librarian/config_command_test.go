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
	"io"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestConfigCommand_Get tests that the config get command successfully reads a value from librarian.yaml.
func TestConfigCommand_Get(t *testing.T) {
	for _, test := range []struct {
		name        string
		args        []string
		configYAML  string
		wantOutput  string
	}{
		{
			name:       "get string value",
			args:       []string{"librarian", "config", "get", "version"},
			configYAML: "version: 1.2.3\n",
			wantOutput: "1.2.3\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			t.Chdir(tempDir)

			if err := os.WriteFile("librarian.yaml", []byte(test.configYAML), 0644); err != nil {
				t.Fatal(err)
			}

			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := Run(t.Context(), test.args...)

			w.Close()
			os.Stdout = oldStdout
			var buf bytes.Buffer
			io.Copy(&buf, r)
			r.Close()

			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.wantOutput, buf.String()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestConfigCommand_Get_Error tests that the config get command returns an error when the path is missing or the key is not found.
func TestConfigCommand_Get_Error(t *testing.T) {
	for _, test := range []struct {
		name        string
		args        []string
		configYAML  string
		wantErrStr  string
	}{
		{
			name:       "missing path",
			args:       []string{"librarian", "config", "get"},
			configYAML: "version: 1.2.3\n",
			wantErrStr: "path is required",
		},
		{
			name:       "key not found",
			args:       []string{"librarian", "config", "get", "foo"},
			configYAML: "version: 1.2.3\n",
			wantErrStr: "unsupported config path",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			t.Chdir(tempDir)

			if err := os.WriteFile("librarian.yaml", []byte(test.configYAML), 0644); err != nil {
				t.Fatal(err)
			}

			err := Run(t.Context(), test.args...)

			if err == nil {
				t.Fatal("expected error; got nil")
			}

			if !strings.Contains(err.Error(), test.wantErrStr) {
				t.Fatalf("got error %v, want containing %q", err, test.wantErrStr)
			}
		})
	}
}

// TestConfigCommand_Set tests that the config set command successfully updates a value in librarian.yaml.
func TestConfigCommand_Set(t *testing.T) {
	for _, test := range []struct {
		name        string
		args        []string
		configYAML  string
		wantYAML    string
	}{
		{
			name:       "set string value",
			args:       []string{"librarian", "config", "set", "version", "1.2.4"},
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

			err := Run(t.Context(), test.args...)

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
		name        string
		args        []string
		configYAML  string
		wantErrStr  string
	}{
		{
			name:       "missing path",
			args:       []string{"librarian", "config", "set"},
			configYAML: "version: 1.2.3\n",
			wantErrStr: "path is required",
		},
		{
			name:       "missing value",
			args:       []string{"librarian", "config", "set", "version"},
			configYAML: "version: 1.2.3\n",
			wantErrStr: "value is required",
		},
		{
			name:       "unsupported path",
			args:       []string{"librarian", "config", "set", "foo", "bar"},
			configYAML: "version: 1.2.3\n",
			wantErrStr: "unsupported config path",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			oldDir, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Chdir(tempDir); err != nil {
				t.Fatal(err)
			}
			defer os.Chdir(oldDir)

			if err := os.WriteFile("librarian.yaml", []byte(test.configYAML), 0644); err != nil {
				t.Fatal(err)
			}

			err = Run(t.Context(), test.args...)

			if err == nil {
				t.Fatal("expected error; got nil")
			}

			if !strings.Contains(err.Error(), test.wantErrStr) {
				t.Fatalf("got error %v, want containing %q", err, test.wantErrStr)
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

	err := Run(t.Context(), "librarian", "config", "set", "version", "1.2.4")

	if err == nil {
		t.Fatal("expected error; got nil")
	}

	if !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("got error %v, want containing %q", err, "permission denied")
	}
}
