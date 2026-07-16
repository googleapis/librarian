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
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

func stubExecutables(t *testing.T) {
	t.Helper()
	if os.Getenv("LIBRARIAN_CACHE") == "" {
		t.Setenv("LIBRARIAN_CACHE", t.TempDir())
	}
	if os.Getenv("LIBRARIAN_BIN") == "" {
		t.Setenv("LIBRARIAN_BIN", t.TempDir())
	}
	bin := t.TempDir()
	pnpmStub := `#!/bin/sh
if [ "$1" = "--version" ]; then
    echo "7.32.2"
    exit 0
fi

# Assert that transient environmental variables are set dynamically for process lifetime
if [ -n "$PNPM_HOME" ] && \
   { [ -n "$PNPM_CONFIG_GLOBAL_BIN_DIR" ] || [ -n "$NPM_CONFIG_GLOBAL_BIN_DIR" ]; } && \
   { [ -n "$PNPM_CONFIG_GLOBAL_DIR" ] || [ -n "$NPM_CONFIG_GLOBAL_DIR" ]; } && \
   { [ -n "$PNPM_CONFIG_STORE_DIR" ] || [ -n "$NPM_CONFIG_STORE_DIR" ]; } && \
   [ -n "$NPM_CONFIG_CACHE" ] && \
   [ -n "$PNPM_CONFIG_DANGEROUSLY_ALLOW_ALL_BUILDS" ]; then
    :
else
    echo "Error: Required transient PNPM/NPM environment variables are missing!" >&2
    exit 1
fi

if [ "$PNPM_ADD_FAIL" = "1" ]; then
    echo "pnpm add error" >&2
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
	corepackStub := `#!/bin/sh
if [ -z "$COREPACK_HOME" ] || [ -z "$COREPACK_ENABLE_DOWNLOAD_PROMPT" ]; then
    echo "Error: Required Corepack environment variables missing!" >&2
    exit 1
fi
if [ "$COREPACK_STUB_FAIL" = "1" ]; then
    echo "Corepack stub error triggered" >&2
    exit 1
fi
case "$*" in
    *install*pnpm@fail*)
        echo "Corepack install failure triggered" >&2
        exit 1
        ;;
    *enable*--install-directory*)
        dir=""
        prev=""
        for arg in "$@"; do
            if [ "$prev" = "--install-directory" ]; then
                dir="$arg"
                break
            fi
            prev="$arg"
        done
        if [ -n "$dir" ]; then
            mkdir -p "$dir"
            printf '#!/bin/sh\nif [ "$1" = "--version" ]; then\n  echo "8.0.0"\n  exit 0\nfi\ncase "$*" in\n  *install*)\n    mkdir -p node_modules/.bin\n    printf "#!/bin/sh\\nmkdir -p build\\n" > node_modules/.bin/tsc\n    chmod +x node_modules/.bin/tsc\n    ;;\nesac\nexit 0\n' > "$dir/pnpm"
            chmod +x "$dir/pnpm"
        fi
        ;;
esac
exit 0
`
	if err := os.WriteFile(filepath.Join(bin, "pnpm"), []byte(pnpmStub), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "node"), []byte(nodeStub), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "corepack"), []byte(corepackStub), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func stubExecutablesWithoutPNPM(t *testing.T) {
	t.Helper()
	if os.Getenv("LIBRARIAN_CACHE") == "" {
		t.Setenv("LIBRARIAN_CACHE", t.TempDir())
	}
	if os.Getenv("LIBRARIAN_BIN") == "" {
		t.Setenv("LIBRARIAN_BIN", t.TempDir())
	}
	bin := t.TempDir()
	nodeStub := `#!/bin/sh
exit 0
`
	corepackStub := `#!/bin/sh
if [ -z "$COREPACK_HOME" ] || [ -z "$COREPACK_ENABLE_DOWNLOAD_PROMPT" ]; then
    echo "Error: Required Corepack environment variables missing!" >&2
    exit 1
fi
case "$*" in
    *enable*--install-directory*)
        dir=""
        prev=""
        for arg in "$@"; do
            if [ "$prev" = "--install-directory" ]; then
                dir="$arg"
                break
            fi
            prev="$arg"
        done
        if [ -n "$dir" ]; then
            mkdir -p "$dir"
            printf '#!/bin/sh\nif [ "$1" = "--version" ]; then\n  echo "7.32.2"\n  exit 0\nfi\ncase "$*" in\n  *install*)\n    mkdir -p node_modules/.bin\n    printf "#!/bin/sh\\nmkdir -p build\\n" > node_modules/.bin/tsc\n    chmod +x node_modules/.bin/tsc\n    ;;\nesac\nexit 0\n' > "$dir/pnpm"
            chmod +x "$dir/pnpm"
        fi
        ;;
esac
exit 0
`
	if err := os.WriteFile(filepath.Join(bin, "node"), []byte(nodeStub), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "corepack"), []byte(corepackStub), 0o755); err != nil {
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
				PNPMVersion: "7.32.2",
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
			name: "non-build tool",
			tools: &config.Tools{
				PNPMVersion: "7.32.2",
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
		{
			name: "tool configuration triggers corepack bootstrap on version mismatch",
			tools: &config.Tools{
				PNPMVersion: "8.0.0",
				PNPM: []*config.PNPMTool{
					{
						Name:    "gapic-node-processing",
						Version: "0.1.8",
					},
				},
			},
			setup: func(t *testing.T) {
				t.Setenv("LIBRARIAN_CACHE", t.TempDir())
				t.Setenv("LIBRARIAN_BIN", t.TempDir())
				stubExecutables(t)
			},
		},
		{
			name: "tool configuration triggers corepack bootstrap on missing pnpm",
			tools: &config.Tools{
				PNPM: []*config.PNPMTool{
					{
						Name:    "gapic-node-processing",
						Version: "0.1.8",
					},
				},
			},
			setup: func(t *testing.T) {
				t.Setenv("LIBRARIAN_CACHE", t.TempDir())
				t.Setenv("LIBRARIAN_BIN", t.TempDir())
				stubExecutablesWithoutPNPM(t)
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
			name:  "missing node or corepack in path",
			tools: &config.Tools{PNPMVersion: "7.32.2", PNPM: []*config.PNPMTool{{Name: "foo", Version: "1.0"}}},
			setup: func(t *testing.T) {
				t.Setenv("PATH", t.TempDir())
			},
			wantErr: errMissingExecutable,
		},
		{
			name: "missing package url for build tool",
			tools: &config.Tools{
				PNPMVersion: "7.32.2",
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
				PNPMVersion: "7.32.2",
				PNPM: []*config.PNPMTool{
					{Name: "tool", Package: "invalid-url", Build: []string{"echo 1"}},
				},
			},
			setup: func(t *testing.T) {
				stubExecutables(t)
			},
			wantErr: errCannotExtractRepo,
		},
		{
			name: "missing pnpm version and missing pnpm in PATH",
			tools: &config.Tools{
				PNPM: []*config.PNPMTool{
					{Name: "tool", Version: "1.0"},
				},
			},
			setup: func(t *testing.T) {
				t.Setenv("PATH", t.TempDir())
			},
			wantErr: errMissingExecutable,
		},
		{
			name: "empty pnpm version and missing pnpm in PATH",
			tools: &config.Tools{
				PNPMVersion: "",
				PNPM: []*config.PNPMTool{
					{Name: "tool", Version: "1.0"},
				},
			},
			setup: func(t *testing.T) {
				t.Setenv("PATH", t.TempDir())
			},
			wantErr: errMissingExecutable,
		},
		{
			name: "corepack enable failure",
			tools: &config.Tools{
				PNPMVersion: "8.0.0",
				PNPM: []*config.PNPMTool{
					{Name: "tool", Version: "1.0"},
				},
			},
			setup: func(t *testing.T) {
				t.Setenv("LIBRARIAN_CACHE", t.TempDir())
				t.Setenv("LIBRARIAN_BIN", t.TempDir())
				t.Setenv("COREPACK_STUB_FAIL", "1")
				stubExecutables(t)
			},
			wantErr: nil, // checked via non-nil error
		},
		{
			name: "corepack install failure",
			tools: &config.Tools{
				PNPMVersion: "fail",
				PNPM: []*config.PNPMTool{
					{Name: "tool", Version: "1.0"},
				},
			},
			setup: func(t *testing.T) {
				t.Setenv("LIBRARIAN_CACHE", t.TempDir())
				t.Setenv("LIBRARIAN_BIN", t.TempDir())
				stubExecutables(t)
			},
			wantErr: nil,
		},
		{
			name: "pnpm add global failure",
			tools: &config.Tools{
				PNPMVersion: "7.32.2",
				PNPM: []*config.PNPMTool{
					{Name: "tool", Version: "1.0"},
				},
			},
			setup: func(t *testing.T) {
				t.Setenv("LIBRARIAN_CACHE", t.TempDir())
				t.Setenv("LIBRARIAN_BIN", t.TempDir())
				t.Setenv("PNPM_ADD_FAIL", "1")
				stubExecutables(t)
			},
			wantErr: nil,
		},
		{
			name: "source build command failure",
			tools: &config.Tools{
				PNPMVersion: "7.32.2",
				PNPM: []*config.PNPMTool{
					{
						Name:    "gapic-generator-typescript",
						Version: "4.12.1",
						Package: "https://github.com/googleapis/google-cloud-node/archive/gapic-generator-v4.12.1.tar.gz",
						Build: []string{
							"exit 1",
						},
					},
				},
			},
			setup: func(t *testing.T) {
				cache := t.TempDir()
				t.Setenv("LIBRARIAN_CACHE", cache)
				t.Setenv("LIBRARIAN_BIN", t.TempDir())
				genDir := filepath.Join(cache,
					"github.com/googleapis/google-cloud-node@4.12.1",
					gapicGeneratorSubdir)
				if err := os.MkdirAll(genDir, 0o755); err != nil {
					t.Fatal(err)
				}
				stubExecutables(t)
			},
			wantErr: nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.setup != nil {
				test.setup(t)
			}
			err := Install(t.Context(), test.tools)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("Install() error = %v, wantErr = %v", err, test.wantErr)
				}
			} else if err == nil {
				t.Fatalf("Install() expected error, got nil")
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

func TestIsPNPMInstalled(t *testing.T) {
	for _, test := range []struct {
		name    string
		version string
		setup   func(t *testing.T)
		want    bool
	}{
		{
			name:    "version matches stub",
			version: "7.32.2",
			setup:   stubExecutables,
			want:    true,
		},
		{
			name:    "empty version requested and pnpm present",
			version: "",
			setup:   stubExecutables,
			want:    true,
		},
		{
			name:    "version mismatch",
			version: "9.9.9",
			setup:   stubExecutables,
			want:    false,
		},
		{
			name:    "pnpm missing in PATH",
			version: "7.32.2",
			setup: func(t *testing.T) {
				t.Setenv("PATH", t.TempDir())
			},
			want: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.setup != nil {
				test.setup(t)
			}
			env, err := getPNPMEnv()
			if err != nil {
				t.Fatal(err)
			}
			got := isPNPMInstalled(t.Context(), env, test.version)
			if got != test.want {
				t.Errorf("isPNPMInstalled(version=%q) = %t, want %t", test.version, got, test.want)
			}
		})
	}
}

func TestGetPNPMEnv(t *testing.T) {
	for _, test := range []struct {
		name string
	}{
		{
			name: "returns valid environment slice",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("LIBRARIAN_CACHE", t.TempDir())
			t.Setenv("LIBRARIAN_BIN", t.TempDir())
			env, err := getPNPMEnv()
			if err != nil {
				t.Fatal(err)
			}
			if len(env) == 0 {
				t.Errorf("getPNPMEnv() returned empty slice")
			}
		})
	}
}
