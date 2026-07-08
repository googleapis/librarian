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

package dart

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestPublish(t *testing.T) {
	tmp := t.TempDir()
	outputA := filepath.Join(tmp, "packages", "lib_a")
	outputB := filepath.Join(tmp, "packages", "lib_b")
	outputC := filepath.Join(tmp, "packages", "lib_c")

	if err := os.MkdirAll(outputA, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outputB, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outputC, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Language: "dart",
		Default: &config.Default{
			Output: filepath.Join(tmp, "packages"),
		},
		Libraries: []*config.Library{
			{
				Name:   "lib_a",
				Output: outputA,
			},
			{
				Name:   "lib_b",
				Output: outputB,
				Dart: &config.DartPackage{
					Packages: map[string]string{
						"package:lib_a": "^0.4.0",
					},
				},
			},
			{
				Name:   "lib_c",
				Output: outputC,
				Dart: &config.DartPackage{
					Packages: map[string]string{
						"package:lib_b": "^0.4.0",
					},
				},
			},
		},
	}

	// Create mock dart executable
	binDir := t.TempDir()
	mockLogFile := filepath.Join(t.TempDir(), "mock_dart.log")
	script := fmt.Sprintf("#!/bin/sh\necho \"dart $* in $(pwd)\" >> \"%s\"\n", mockLogFile)
	if err := os.WriteFile(filepath.Join(binDir, "dart"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	tests := []struct {
		name               string
		librariesToPublish []string
		execute            bool
		wantLogs           []string
	}{
		{
			name:               "dry run all packages",
			librariesToPublish: []string{"lib_c", "lib_a", "lib_b"},
			execute:            false,
			wantLogs: []string{
				fmt.Sprintf("dart pub publish --dry-run in %s", outputA),
				fmt.Sprintf("dart pub publish --dry-run in %s", outputB),
				fmt.Sprintf("dart pub publish --dry-run in %s", outputC),
			},
		},
		{
			name:               "execute subset of packages",
			librariesToPublish: []string{"lib_c", "lib_a"},
			execute:            true,
			wantLogs: []string{
				fmt.Sprintf("dart pub publish --force in %s", outputA),
				fmt.Sprintf("dart pub publish --force in %s", outputC),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := os.Remove(mockLogFile); err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}

			err := Publish(t.Context(), cfg, test.librariesToPublish, test.execute)
			if err != nil {
				t.Fatalf("Publish failed: %v", err)
			}

			content, err := os.ReadFile(mockLogFile)
			if err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}

			var gotLogs []string
			if len(content) > 0 {
				gotLogs = strings.Split(strings.TrimSpace(string(content)), "\n")
			}

			if diff := cmp.Diff(test.wantLogs, gotLogs); diff != "" {
				t.Errorf("invocations mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
