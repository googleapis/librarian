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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/cache"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
	"github.com/googleapis/librarian/internal/tool/pip"
)

func TestInstall(t *testing.T) {
	setupStubPip(t, "#!/bin/sh\n")
	tools := &config.Tools{
		Pip: []*config.PipTool{
			{Name: "ruff", Version: "0.14.14"},
		},
	}
	if err := Install(t.Context(), tools); err != nil {
		t.Fatal(err)
	}
}

func TestInstall_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		tools   *config.Tools
		setup   func(t *testing.T)
		wantErr error
	}{
		{
			name:    "nil tools config",
			tools:   nil,
			wantErr: ErrNoToolsSpecified,
		},
		{
			name:    "empty pip tools",
			tools:   &config.Tools{},
			wantErr: ErrNoToolsSpecified,
		},
		{
			name: "local path not found",
			tools: &config.Tools{
				Pip: []*config.PipTool{
					{LocalPath: "/path/does/not/exist"},
				},
			},
			wantErr: pip.ErrLocalPathNotFound,
		},
		{
			name: "pip execution fails",
			tools: &config.Tools{
				Pip: []*config.PipTool{
					{Name: "invalid-tool"},
				},
			},
			setup: func(t *testing.T) {
				setupStubPip(t, "#!/bin/sh\nexit 1\n")
			},
			wantErr: pip.ErrInstall,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.setup != nil {
				test.setup(t)
			}
			err := Install(t.Context(), test.tools)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Install() error = %v, wantErr = %v", err, test.wantErr)
			}
		})
	}
}

func TestInstallDir(t *testing.T) {
	tmpDir := t.TempDir()
	for _, test := range []struct {
		name           string
		librarianBin   string
		librarianCache string
		want           string
	}{
		{
			name:         "LIBRARIAN_BIN is set",
			librarianBin: tmpDir,
			want:         filepath.Join(tmpDir, "python_tools"),
		},
		{
			name:           "LIBRARIAN_CACHE is set",
			librarianCache: tmpDir,
			want:           filepath.Join(tmpDir, "bin", "python_tools"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(cache.EnvLibrarianBin, test.librarianBin)
			t.Setenv(cache.EnvLibrarianCache, test.librarianCache)
			got, err := InstallDir()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTemplateDirectory(t *testing.T) {
	binDir := t.TempDir()
	t.Setenv(cache.EnvLibrarianBin, binDir)
	got, err := templateDirectory()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(binDir, "python_tools", "templates")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestExtractTemplates(t *testing.T) {
	binDir := t.TempDir()
	t.Setenv(cache.EnvLibrarianBin, binDir)
	if err := extractTemplates(); err != nil {
		t.Fatal(err)
	}
	wantDir := filepath.Join(binDir, "python_tools", "templates", "python_mono_repo_library")
	for _, file := range []string{
		"README.rst",
		"docs/index.rst",
		"docs/summary_overview.md",
	} {
		path := filepath.Join(wantDir, file)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected template file %s to exist: %v", path, err)
		}
	}
}

func setupStubPip(t *testing.T, script string) {
	t.Helper()
	bin := t.TempDir()
	testhelper.WriteExecutable(t, filepath.Join(bin, "pip"), script)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv(cache.EnvLibrarianBin, t.TempDir())
}

func TestBinDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(cache.EnvLibrarianBin, tmpDir)
	got, err := binDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(tmpDir, "python_tools", "bin")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestPandocPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(cache.EnvLibrarianBin, tmpDir)
	got, err := pandocPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(tmpDir, "python_tools", "bin", "pandoc")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestToolsEnv(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(cache.EnvLibrarianBin, tmpDir)
	expectedBin := filepath.Join(tmpDir, "python_tools", "bin")
	expectedPandoc := filepath.Join(expectedBin, "pandoc")

	t.Run("without pandoc binary", func(t *testing.T) {
		got, err := toolsEnv()
		if err != nil {
			t.Fatal(err)
		}
		want := map[string]string{
			"PATH": expectedBin,
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("with pandoc binary", func(t *testing.T) {
		if err := os.MkdirAll(expectedBin, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(expectedPandoc, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		defer os.Remove(expectedPandoc)

		got, err := toolsEnv()
		if err != nil {
			t.Fatal(err)
		}
		want := map[string]string{
			"PATH":            expectedBin,
			"PYPANDOC_PANDOC": expectedPandoc,
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("with user env override", func(t *testing.T) {
		t.Setenv("PYPANDOC_PANDOC", "/custom/pandoc")
		got, err := toolsEnv()
		if err != nil {
			t.Fatal(err)
		}
		want := map[string]string{
			"PATH":            expectedBin,
			"PYPANDOC_PANDOC": "/custom/pandoc",
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestInstall_PandocBinary(t *testing.T) {
	stubScript := `#!/bin/sh
target=""
while [ $# -gt 0 ]; do
  if [ "$1" = "--target" ]; then
    target="$2"
    shift 2
  else
    shift
  fi
done

if [ -n "$target" ]; then
  mkdir -p "$target/pypandoc/files"
  echo "pandoc binary content" > "$target/pypandoc/files/pandoc"
fi
exit 0
`
	setupStubPip(t, stubScript)

	tools := &config.Tools{
		Pip: []*config.PipTool{
			{Name: "pypandoc-binary", Version: "1.16.2"},
		},
	}
	if err := Install(t.Context(), tools); err != nil {
		t.Fatal(err)
	}

	p, err := pandocPath()
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("expected pandoc binary to exist at %s: %v", p, err)
	}
	if !strings.Contains(string(content), "pandoc binary content") {
		t.Errorf("unexpected content: %s", string(content))
	}

	// Test idempotency / cached version check
	if err := Install(t.Context(), tools); err != nil {
		t.Fatalf("second Install call failed: %v", err)
	}
}
