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

package ruby

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestFormat(t *testing.T) {
	binDir := t.TempDir()
	t.Setenv("LIBRARIAN_BIN", binDir)
	if err := Format(t.Context()); err != nil {
		t.Fatal(err)
	}
	installDir := filepath.Join(binDir, "ruby_tools")
	configPath := filepath.Join(installDir, "gems", "gapic-generator-cloud-0.52.0", "cloud-rubocop.yml")
	for _, dir := range []string{filepath.Dir(configPath), filepath.Join(installDir, "bin")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(configPath, []byte("AllCops:\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testhelper.WriteExecutable(t, filepath.Join(installDir, "bin", "rubocop"), `#!/bin/sh
while [ $# -gt 0 ]; do
  case "$1" in
    --cache|--fail-level|-c) shift 2 ;;
    -*) shift ;;
    *) echo "# formatted" >> "$1"; shift ;;
  esac
done
`)
	outDir := t.TempDir()
	want := map[string]string{
		"foo/.toys.rb":           "x = 1\n# formatted\n",
		"foo/Gemfile":            "x = 1\n# formatted\n",
		"foo/snippets/Gemfile":   "x = 1\n# formatted\n",
		"foo/lib/foo.rb":         "x = 1\n# formatted\n",
		"foo/proto_docs/foo.rb":  "x = 1\n# formatted\n",
		"foo/snippets/foo.rb":    "x = 1\n# formatted\n",
		"foo/lib/foo_pb.rb":      "x = 1\n",
		"foo/lib/kept.rb":        "x = 1\n",
		"foo/lib/foo/version.rb": "x = 1\n",
		"foo/test/foo_test.rb":   "x = 1\n",
		"bar/lib/bar.rb":         "x = 1\n# formatted\n",
	}
	for relPath := range want {
		filePath := filepath.Join(outDir, relPath)
		if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filePath, []byte("x = 1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	libraries := []*config.Library{
		{Name: "foo", Output: filepath.Join(outDir, "foo"), Keep: []string{"lib/kept.rb"}},
		{Name: "bar", Output: filepath.Join(outDir, "bar")},
	}
	if err := Format(t.Context(), libraries...); err != nil {
		t.Fatal(err)
	}
	got := make(map[string]string)
	for relPath := range want {
		content, err := os.ReadFile(filepath.Join(outDir, relPath))
		if err != nil {
			t.Fatal(err)
		}
		got[relPath] = string(content)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFormat_Error(t *testing.T) {
	t.Setenv("LIBRARIAN_BIN", t.TempDir())
	outDir := t.TempDir()
	filePath := filepath.Join(outDir, "lib", "foo.rb")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte("x = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name    string
		output  string
		wantErr error
	}{
		{
			name:    "missing output directory",
			output:  filepath.Join(outDir, "nonexistent"),
			wantErr: fs.ErrNotExist,
		},
		{
			name:    "missing rubocop config",
			output:  outDir,
			wantErr: ErrRubocopConfigNotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotErr := Format(t.Context(), &config.Library{Name: "foo", Output: test.output})
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("Format() error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
	}
}
