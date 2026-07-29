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

package composer

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

func TestInstall(t *testing.T) {
	testhelper.RequireCommand(t, "composer")
	cache := t.TempDir()
	t.Setenv("LIBRARIAN_CACHE", cache)
	repoDir := filepath.Join(cache, "github.com/googleapis/gapic-generator-php@1.0.0")
	if err := os.MkdirAll(filepath.Join(repoDir, "dummy"), 0o755); err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	testhelper.WriteExecutable(t, filepath.Join(bin, "composer"), "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	binDir := t.TempDir()
	tools := []*config.ComposerTool{
		{
			Name:    "gapic-generator-php",
			Version: "1.0.0",
			Repo:    "github.com/googleapis/gapic-generator-php",
			SHA256:  "29635b02c6e505fe31cba2f88ae999f00d2710fe1d65cb7cad521a82e7c5a518",
		},
	}
	if err := Install(t.Context(), tools, "php", binDir); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	wrapperPath := filepath.Join(binDir, "gapic-generator-php")
	b, err := os.ReadFile(wrapperPath)
	if err != nil {
		t.Fatal(err)
	}
	destPath := filepath.Join(repoDir, "src", "Main.php")
	want := phpWrapperContent("php", destPath)
	if diff := cmp.Diff(want, string(b)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestInstall_Error(t *testing.T) {
	binDir := t.TempDir()
	tools := []*config.ComposerTool{
		{
			Name:    "",
			Version: "1.0.0",
		},
	}
	gotErr := Install(t.Context(), tools, "php", binDir)
	if !errors.Is(gotErr, ErrInvalidTool) {
		t.Fatalf("Install() error = %v, wantErr = %v", gotErr, ErrInvalidTool)
	}
}

func TestCreateBinWrapper(t *testing.T) {
	for _, test := range []struct {
		name        string
		wrapperName string
	}{
		{
			name:        "simple wrapper",
			wrapperName: "foo",
		},
		{
			name:        "nested wrapper",
			wrapperName: "nested/dir/foo",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			binDir := t.TempDir()
			destPath := "/path/to/dest"
			content := fmt.Sprintf("#!/bin/sh\nexec %q \"$@\"\n", destPath)
			if err := createBinWrapper(test.wrapperName, content, binDir); err != nil {
				t.Fatal(err)
			}
			wrapperPath := filepath.Join(binDir, test.wrapperName)
			b, err := os.ReadFile(wrapperPath)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(content, string(b)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			info, err := os.Stat(wrapperPath)
			if err != nil {
				t.Fatal(err)
			}
			perm := info.Mode().Perm()
			if perm&0o700 != 0o700 {
				t.Errorf("wrapper permissions = %04o, want at least 0700 (rwx) for owner", perm)
			}
			if perm&0o022 != 0 {
				t.Errorf("wrapper should not be writable by group/others: %04o", perm)
			}
		})
	}
}

func TestVerify(t *testing.T) {
	tools := []*config.ComposerTool{
		{Name: "gapic-generator-php", Version: "1.0.0", Repo: "github.com/googleapis/gapic-generator-php"},
	}
	if err := verify(tools); err != nil {
		t.Errorf("verify() error = %v, want nil", err)
	}
}

func TestVerify_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		tools   []*config.ComposerTool
		wantErr error
	}{
		{
			name: "missing name",
			tools: []*config.ComposerTool{
				{Name: "", Version: "1.0.0", Repo: "github.com"},
			},
			wantErr: ErrInvalidTool,
		},
		{
			name: "missing version",
			tools: []*config.ComposerTool{
				{Name: "gapic-generator-php", Version: "", Repo: "github.com"},
			},
			wantErr: ErrInvalidTool,
		},
		{
			name: "missing repo",
			tools: []*config.ComposerTool{
				{Name: "gapic-generator-php", Version: "1.0.0", Repo: ""},
			},
			wantErr: ErrMissingRepo,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotErr := verify(test.tools)
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("verify() error = %v, wantErr = %v", gotErr, test.wantErr)
			}
		})
	}
}
