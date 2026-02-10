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

package python

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestBump(t *testing.T) {
	const readme = "This is a readme file"
	const versionBefore = "line1\n" + gapicVersionLinePrefix + "\"1.2.2\"# Other stuff\n" + "line3"
	const versionAfter = "line1\n" + gapicVersionLinePrefix + "\"1.2.3\"\n" + "line3"
	initial := map[string]string{
		"README.txt":                              readme,
		"docs/README.txt":                         readme,
		"google/cloud/iam/" + gapicVersionFile:    versionBefore,
		"google/cloud/iam_v1/" + gapicVersionFile: versionBefore,
		"other/" + gapicVersionFile:               versionBefore,
	}
	dir := t.TempDir()
	for file, content := range initial {
		path := filepath.Join(dir, file)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := Bump(dir, "1.2.3"); err != nil {
		t.Fatal(err)
	}

	wantAfter := map[string]string{
		"README.txt":                           readme,
		"docs/README.txt":                      readme,
		"google/cloud/iam/gapic_version.py":    versionAfter,
		"google/cloud/iam_v1/gapic_version.py": versionAfter,
		"other/gapic_version.py":               versionAfter,
	}
	for file, want := range wantAfter {
		got, err := os.ReadFile(filepath.Join(dir, file))
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(want, string(got)); diff != "" {
			t.Errorf("mismatch in file %s (-want +got):\n%s", file, diff)
		}
	}
}

func TestBump_Error(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		setup   func(string) error
		wantErr error
	}{
		{
			name: "gapic_version.py is a symlink",
			setup: func(dir string) error {
				textPath := filepath.Join(dir, "test.txt")
				versionPath := filepath.Join(dir, gapicVersionFile)
				if err := os.WriteFile(textPath, []byte("just a text file"), 0644); err != nil {
					return err
				}
				return os.Symlink(textPath, versionPath)
			},
			wantErr: errSymLinkVersionFile,
		},
		{
			name: "gapic_version.py has no version",
			setup: func(dir string) error {
				versionPath := filepath.Join(dir, gapicVersionFile)
				return os.WriteFile(versionPath, []byte("no version here"), 0644)
			},
			wantErr: errNoVersionFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := test.setup(dir); err != nil {
				t.Fatal(err)
			}
			gotErr := Bump(dir, "1.2.3")
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("bumpSingleGapicVersion() error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
	}
}

func TestBumpSingleGapicVersionFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), gapicVersionFile)
	initialText :=
		`line1
__version__ = "before" # irrelevant
line3`
	if err := os.WriteFile(path, []byte(initialText), 0644); err != nil {
		t.Fatal(err)
	}
	if err := bumpSingleGapicVersionFile(path, "1.2.3"); err != nil {
		t.Fatal(err)
	}
	want :=
		`line1
__version__ = "1.2.3"
line3`
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, string(got)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestBumpSingleGapicVersionFile_Error(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		setup   func(string) error
		wantErr error
	}{
		{
			name: "file doesn't exist",
			setup: func(string) error {
				return nil
			},
			wantErr: os.ErrNotExist,
		},
		{
			name: "multiple version lines",
			setup: func(path string) error {
				content := fmt.Sprintf("line1\n%s\"1.2.3\"\n%s\"4.5.6\"", gapicVersionLinePrefix, gapicVersionLinePrefix)
				return os.WriteFile(path, []byte(content), 0644)
			},
			wantErr: errMultipleVersions,
		},
		{
			name: "no version line",
			setup: func(path string) error {
				return os.WriteFile(path, []byte("line1\nline2"), 0644)
			},
			wantErr: errNoVersionFound,
		},
		{
			name: "can't write to file",
			setup: func(path string) error {
				content := fmt.Sprintf("line1\n%s\"1.2.3\"\nline3", gapicVersionLinePrefix)
				// 0444 is read-only, even for the owner
				return os.WriteFile(path, []byte(content), 0444)
			},
			wantErr: os.ErrPermission,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), gapicVersionFile)
			if err := test.setup(path); err != nil {
				t.Fatal(err)
			}
			gotErr := bumpSingleGapicVersionFile(path, "1.2.3")
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("bumpSingleGapicVersion() error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
	}
}
