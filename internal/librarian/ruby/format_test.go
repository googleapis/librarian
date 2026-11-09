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

// TestFormat_NoLibraries verifies that Format is a no-op when no libraries are passed.
func TestFormat_NoLibraries(t *testing.T) {
	testFormatLibraries(t, 0)
}

// TestFormat_SingleLibrary verifies that Format formats all eligible Ruby files
// across multi-API directories while preserving kept and excluded files.
func TestFormat_SingleLibrary(t *testing.T) {
	testFormatLibraries(t, 1)
}

// TestFormat_MultipleLibraries verifies that Format formats the same set of
// Ruby files identically across multiple libraries in a single batch call.
func TestFormat_MultipleLibraries(t *testing.T) {
	testFormatLibraries(t, 4)
}

// TestFormat_MultipleBatches verifies that Format chunks and formats files
// across multiple rubocop invocations when total files exceed maxFilesPerFormatBatch.
func TestFormat_MultipleBatches(t *testing.T) {
	testFormatLibraries(t, 26)
}

func TestFormat_MissingOutputDir_Error(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "nonexistent")
	tools := &config.Tools{
		Gem: []*config.GemTool{{Name: generatorGemName, Version: "0.55.0"}},
	}
	gotErr := Format(t.Context(), tools, &config.Library{Name: "foo", Output: outDir})
	if !errors.Is(gotErr, fs.ErrNotExist) {
		t.Errorf("Format() error = %v, wantErr %v", gotErr, fs.ErrNotExist)
	}
}

func TestFormat_MissingRubocopConfig_Error(t *testing.T) {
	for _, test := range []struct {
		name  string
		tools *config.Tools
	}{
		{
			name:  "nil tools",
			tools: nil,
		},
		{
			name: "missing generator gem in tools",
			tools: &config.Tools{
				Gem: []*config.GemTool{{Name: "toys", Version: "0.22.0"}},
			},
		},
		{
			name: "installed gem config file does not exist",
			tools: &config.Tools{
				Gem: []*config.GemTool{{Name: generatorGemName, Version: "0.55.0"}},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("LIBRARIAN_BIN", t.TempDir())
			outDir := t.TempDir()
			filePath := filepath.Join(outDir, "lib", "foo.rb")
			if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filePath, []byte("x = 1\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			gotErr := Format(t.Context(), test.tools, &config.Library{Name: "foo", Output: outDir})
			if !errors.Is(gotErr, ErrRubocopConfigNotFound) {
				t.Errorf("Format() error = %v, wantErr %v", gotErr, ErrRubocopConfigNotFound)
			}
		})
	}
}

func testFormatLibraries(t *testing.T, count int) {
	t.Helper()
	binDir := t.TempDir()
	t.Setenv("LIBRARIAN_BIN", binDir)
	installDir := filepath.Join(binDir, "ruby_tools")
	staleConfigPath := filepath.Join(installDir, "gems", "gapic-generator-cloud-0.52.0", "cloud-rubocop.yml")
	pinnedConfigPath := filepath.Join(installDir, "gems", "gapic-generator-cloud-0.55.0", "cloud-rubocop.yml")
	for _, dir := range []string{filepath.Dir(staleConfigPath), filepath.Dir(pinnedConfigPath), filepath.Join(installDir, "bin")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(staleConfigPath, []byte("# stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pinnedConfigPath, []byte("# formatted\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testhelper.WriteExecutable(t, filepath.Join(installDir, "bin", "rubocop"), `#!/bin/sh
config_file=""
while [ $# -gt 0 ]; do
  case "$1" in
    -c) config_file="$2"; shift 2 ;;
    --cache|--fail-level) shift 2 ;;
    -*) shift ;;
    *) cat "$config_file" >> "$1"; shift ;;
  esac
done
`)
	tools := &config.Tools{
		Gem: []*config.GemTool{
			{Name: generatorGemName, Version: "0.55.0"},
		},
	}
	outDir := t.TempDir()
	var libraries []*config.Library
	want := make(map[string]string)
	for index := range count {
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
	if err := Format(t.Context(), tools, libraries...); err != nil {
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
