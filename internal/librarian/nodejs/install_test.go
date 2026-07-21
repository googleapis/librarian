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

package nodejs

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

func stubExecutables(t *testing.T) {
	t.Helper()
	bin := t.TempDir()
	pnpmStub := `#!/bin/sh
# Assert that transient environmental variables are set dynamically for process lifetime
if [ -n "$PNPM_HOME" ] && [ -n "$PNPM_CONFIG_GLOBAL_BIN_DIR" ] && [ -n "$PNPM_CONFIG_GLOBAL_DIR" ] && [ -n "$PNPM_CONFIG_STORE_DIR" ]; then
    :
else
    echo "Error: Required transient PNPM environment variables are missing!" >&2
    exit 1
fi

case "$*" in
    *install*)
        mkdir -p node_modules/.bin
        printf '#!/bin/sh\nmkdir -p build\n' > node_modules/.bin/tsc
        chmod +x node_modules/.bin/tsc
        ;;
    *add\ -g*)
        ;;
esac
exit 0
`
	nodeStub := `#!/bin/sh
exit 0
`
	if err := os.WriteFile(filepath.Join(bin, "pnpm"), []byte(pnpmStub), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "node"), []byte(nodeStub), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestInstall(t *testing.T) {
	for _, test := range []struct {
		name  string
		tools *config.Tools
		setup func(t *testing.T)
	}{
		{
			name: "source build tool",
			tools: &config.Tools{
				PNPM: []*config.PNPMTool{
					{
						Name:    "gapic-generator-typescript",
						Version: "4.12.1",
						Package: "https://github.com/googleapis/google-cloud-node/archive/gapic-generator-v4.12.1.tar.gz",
						Build: []string{
							"pnpm install",
							"./node_modules/.bin/tsc",
							"cp -a templates protos build/",
						},
					},
				},
			},
			setup: func(t *testing.T) {
				cache := t.TempDir()
				t.Setenv("LIBRARIAN_CACHE", cache)
				binDir := t.TempDir()
				t.Setenv("LIBRARIAN_BIN", binDir)
				genDir := filepath.Join(cache,
					"github.com/googleapis/google-cloud-node@4.12.1",
					gapicGeneratorSubdir)
				for _, sub := range []string{"templates", "protos"} {
					if err := os.MkdirAll(filepath.Join(genDir, sub), 0o755); err != nil {
						t.Fatal(err)
					}
				}
				stubExecutables(t)
			},
		},
		{
			name: "build tool with custom src_dir",
			tools: &config.Tools{
				PNPM: []*config.PNPMTool{
					{
						Name:    "gapic-tools",
						Version: "1.1.0",
						Package: "https://github.com/googleapis/google-cloud-node/archive/gapic-tools-v1.1.0.tar.gz",
						SrcDir:  "core/packages/tools",
						Build:   []string{"true"},
					},
				},
			},
			setup: func(t *testing.T) {
				cache := t.TempDir()
				t.Setenv("LIBRARIAN_CACHE", cache)
				binDir := t.TempDir()
				t.Setenv("LIBRARIAN_BIN", binDir)
				toolsDir := filepath.Join(cache, "github.com/googleapis/google-cloud-node@1.1.0", "core/packages/tools")
				if err := os.MkdirAll(toolsDir, 0o755); err != nil {
					t.Fatal(err)
				}
				stubExecutables(t)
			},
		},
		{
			name: "non-build tool",
			tools: &config.Tools{
				PNPM: []*config.PNPMTool{
					{
						Name:    "gapic-node-processing",
						Version: "0.1.8",
					},
					{
						Name:    "custom-pkg",
						Package: "custom-pkg@1.0.0",
					},
				},
			},
			setup: func(t *testing.T) {
				t.Setenv("LIBRARIAN_CACHE", t.TempDir())
				t.Setenv("LIBRARIAN_BIN", t.TempDir())
				stubExecutables(t)
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.setup != nil {
				test.setup(t)
			}
			if err := Install(t.Context(), test.tools); err != nil {
				t.Fatal(err)
			}
		})
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
			name:    "nil tools",
			tools:   nil,
			wantErr: errNoToolsSpecified,
		},
		{
			name:    "empty tools",
			tools:   &config.Tools{},
			wantErr: errNoToolsSpecified,
		},
		{
			name:  "missing node or pnpm in path",
			tools: &config.Tools{PNPM: []*config.PNPMTool{{Name: "foo", Version: "1.0"}}},
			setup: func(t *testing.T) {
				t.Setenv("PATH", t.TempDir())
			},
			wantErr: errMissingExecutable,
		},
		{
			name: "missing package url for build tool",
			tools: &config.Tools{
				PNPM: []*config.PNPMTool{
					{Name: "tool", Build: []string{"echo 1"}},
				},
			},
			setup: func(t *testing.T) {
				stubExecutables(t)
			},
			wantErr: errMissingPackageURL,
		},
		{
			name: "invalid package url for build tool",
			tools: &config.Tools{
				PNPM: []*config.PNPMTool{
					{Name: "tool", Package: "invalid-url", Build: []string{"echo 1"}},
				},
			},
			setup: func(t *testing.T) {
				stubExecutables(t)
			},
			wantErr: errCannotExtractRepo,
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

func TestRepoFromPackageURL(t *testing.T) {
	for _, test := range []struct {
		name       string
		packageURL string
		want       string
	}{
		{
			name:       "valid archive url",
			packageURL: "https://github.com/googleapis/google-cloud-node/archive/gapic-generator-v4.12.1.tar.gz",
			want:       "github.com/googleapis/google-cloud-node",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := repoFromPackageURL(test.packageURL)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Errorf("repoFromPackageURL(%q) = %q, want %q", test.packageURL, got, test.want)
			}
		})
	}
}

func TestRepoFromPackageURL_Error(t *testing.T) {
	for _, test := range []struct {
		name       string
		packageURL string
		wantErr    error
	}{
		{
			name:       "invalid archive url",
			packageURL: "https://github.com/googleapis/google-cloud-node/invalid",
			wantErr:    errCannotExtractRepo,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := repoFromPackageURL(test.packageURL)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("repoFromPackageURL(%q) error = %v, wantErr = %v", test.packageURL, err, test.wantErr)
			}
		})
	}
}

func TestInstallDir(t *testing.T) {
	for _, test := range []struct {
		name string
	}{
		{
			name: "returns nodejs_tools directory under LIBRARIAN_BIN",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			binDir := t.TempDir()
			t.Setenv("LIBRARIAN_BIN", binDir)
			got, err := InstallDir()
			if err != nil {
				t.Fatal(err)
			}
			want := filepath.Join(binDir, "nodejs_tools")
			if got != want {
				t.Errorf("InstallDir() = %q, want %q", got, want)
			}
		})
	}
}

func TestGetToolsEnv(t *testing.T) {
	for _, test := range []struct {
		name string
	}{
		{
			name: "returns PATH with nodejs_tools bin directory",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			binDir := t.TempDir()
			t.Setenv("LIBRARIAN_BIN", binDir)
			env, err := getToolsEnv()
			if err != nil {
				t.Fatal(err)
			}
			want := filepath.Join(binDir, "nodejs_tools", "bin")
			if got := env["PATH"]; got != want {
				t.Errorf("getToolsEnv()[PATH] = %q, want %q", got, want)
			}
		})
	}
}

func TestGetPNPMEnv(t *testing.T) {
	cacheDir := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("LIBRARIAN_CACHE", cacheDir)
	t.Setenv("LIBRARIAN_BIN", binDir)

	envList, err := getPNPMEnv()
	if err != nil {
		t.Fatal(err)
	}

	wantBinDir := filepath.Join(binDir, "nodejs_tools", "bin")
	wantGlobalDir := filepath.Join(cacheDir, "pnpm-global")
	wantStoreDir := filepath.Join(cacheDir, "pnpm-store")

	envMap := make(map[string]string)
	for _, entry := range envList {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	if got := envMap["PNPM_HOME"]; got != wantBinDir {
		t.Errorf("PNPM_HOME = %q, want %q", got, wantBinDir)
	}
	if got := envMap["PNPM_CONFIG_GLOBAL_BIN_DIR"]; got != wantBinDir {
		t.Errorf("PNPM_CONFIG_GLOBAL_BIN_DIR = %q, want %q", got, wantBinDir)
	}
	if got := envMap["NPM_CONFIG_GLOBAL_BIN_DIR"]; got != wantBinDir {
		t.Errorf("NPM_CONFIG_GLOBAL_BIN_DIR = %q, want %q", got, wantBinDir)
	}
	if got := envMap["npm_config_global_bin_dir"]; got != wantBinDir {
		t.Errorf("npm_config_global_bin_dir = %q, want %q", got, wantBinDir)
	}
	if got := envMap["PNPM_CONFIG_GLOBAL_DIR"]; got != wantGlobalDir {
		t.Errorf("PNPM_CONFIG_GLOBAL_DIR = %q, want %q", got, wantGlobalDir)
	}
	if got := envMap["PNPM_CONFIG_STORE_DIR"]; got != wantStoreDir {
		t.Errorf("PNPM_CONFIG_STORE_DIR = %q, want %q", got, wantStoreDir)
	}
}
