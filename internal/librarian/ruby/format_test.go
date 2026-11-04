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
	"fmt"
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
	for _, test := range []struct {
		name  string
		count int
	}{
		{name: "no libraries", count: 0},
		{name: "single library", count: 1},
		{name: "multiple libraries", count: 4},
		{name: "multiple batches", count: 26},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()
			var libraries []*config.Library
			want := make(map[string]string)
			for index := range test.count {
				libraryName := fmt.Sprintf("foo%d", index)
				libraries = append(libraries, &config.Library{
					Name:   libraryName,
					Output: filepath.Join(outDir, libraryName),
					Keep:   []string{"lib/kept.rb"},
				})
				for relPath, wantContent := range map[string]string{
					".toys.rb":                           "x = 1\n# formatted\n",
					"Gemfile":                            "x = 1\n# formatted\n",
					"snippets/Gemfile":                   "x = 1\n# formatted\n",
					"lib/client.rb":                      "x = 1\n# formatted\n",
					"lib/v1/client.rb":                   "x = 1\n# formatted\n",
					"lib/v2/client.rb":                   "x = 1\n# formatted\n",
					"proto_docs/doc.rb":                  "x = 1\n# formatted\n",
					"snippets/snippet.rb":                "x = 1\n# formatted\n",
					"lib/client_pb.rb":                   "x = 1\n",
					"lib/kept.rb":                        "x = 1\n",
					"lib/" + libraryName + "/version.rb": "x = 1\n",
					"test/client_test.rb":                "x = 1\n",
				} {
					libraryRelPath := filepath.ToSlash(filepath.Join(libraryName, relPath))
					want[libraryRelPath] = wantContent
					filePath := filepath.Join(outDir, libraryRelPath)
					if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filePath, []byte("x = 1\n"), 0o644); err != nil {
						t.Fatal(err)
					}
				}
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
		})
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
