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

package python

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestFormat_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		wantErr error
	}{
		{
			name:    "nil library",
			library: nil,
			wantErr: ErrNilLibrary,
		},
		{
			name: "empty output",
			library: &config.Library{
				Name:   "test-lib",
				Output: "",
			},
			wantErr: ErrEmptyOutput,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := Format(t.Context(), test.library)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("Format(%v) error = %v, wantErr %v", test.library, err, test.wantErr)
			}
		})
	}
}

func TestFormat_Success(t *testing.T) {
	mockDir := t.TempDir()
	mockRuffPath := filepath.Join(mockDir, "ruff")
	logPath := filepath.Join(mockDir, "ruff.log")
	script := fmt.Sprintf("#!/bin/sh\necho \"$@\" >> %q\nexit 0\n", logPath)
	testhelper.WriteExecutable(t, mockRuffPath, script)
	t.Setenv("PATH", mockDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	outDir := t.TempDir()
	absOutDir, err := filepath.Abs(outDir)
	if err != nil {
		t.Fatal(err)
	}

	lib := &config.Library{
		Name:   "test-lib",
		Output: outDir,
	}
	if err := Format(t.Context(), lib); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("format %s\ncheck --fix %s\n", absOutDir, absOutDir)
	if diff := cmp.Diff(want, string(got)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFormat_RuffError(t *testing.T) {
	for _, test := range []struct {
		name       string
		failAction string
	}{
		{
			name:       "ruff format failure",
			failAction: "format",
		},
		{
			name:       "ruff check failure",
			failAction: "check",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			mockDir := t.TempDir()
			script := fmt.Sprintf(`#!/bin/sh
if [ "$1" = "%s" ]; then
	exit 1
fi
exit 0
`, test.failAction)
			testhelper.WriteExecutable(t, filepath.Join(mockDir, "ruff"), script)
			t.Setenv("PATH", mockDir+string(os.PathListSeparator)+os.Getenv("PATH"))

			outDir := t.TempDir()
			lib := &config.Library{
				Name:   "test-lib",
				Output: outDir,
			}
			if err := Format(t.Context(), lib); err == nil {
				t.Errorf("Format(%v) = nil, wantErr on %s failure", lib, test.failAction)
			}
		})
	}
}
