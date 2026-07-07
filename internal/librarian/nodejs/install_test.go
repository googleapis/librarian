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

func TestInstall(t *testing.T) {
	tools := &config.Tools{
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
	}
	tool := tools.PNPM[0]
	repo, err := repoFromPackageURL(tool.Package)
	if err != nil {
		t.Fatal(err)
	}

	// Pre-populate the fetch cache so fetch.Repo returns immediately
	// without downloading the tarball over the network.
	cache := t.TempDir()
	t.Setenv("LIBRARIAN_CACHE", cache)
	binDir := t.TempDir()
	t.Setenv("LIBRARIAN_BIN", binDir)
	genDir := filepath.Join(cache,
		repo+"@"+tool.Version,
		gapicGeneratorSubdir)
	for _, sub := range []string{
		"templates",
		"protos",
	} {
		if err := os.MkdirAll(filepath.Join(genDir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Stub pnpm and node. The pnpm stub also creates
	// node_modules/.bin/tsc in the working directory during 'pnpm install'
	// so the subsequent "./node_modules/.bin/tsc" build step finds an executable.
	bin := t.TempDir()
	pnpmStub := `#!/bin/sh
# Assert that transient environmental variables are set dynamically for process lifetime
if [ -z "$PNPM_HOME" ] || [ -z "$PNPM_CONFIG_GLOBAL_BIN_DIR" ] || [ -z "$PNPM_CONFIG_GLOBAL_DIR" ] || [ -z "$PNPM_CONFIG_STORE_DIR" ] || \
   [ -z "$PNPM_CONFIG_DANGEROUSLY_ALLOW_ALL_BUILDS" ]; then
    echo "Error: Required transient PNPM environment variables are missing!" >&2
    exit 1
fi

case "$*" in
    *install*)
        mkdir -p node_modules/.bin
        printf '#!/bin/sh\nmkdir -p build\n' > node_modules/.bin/tsc
        chmod +x node_modules/.bin/tsc
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

	if err := Install(t.Context(), tools); err != nil {
		t.Fatal(err)
	}
}

func TestInstallDir(t *testing.T) {
	binDir := t.TempDir()
	t.Setenv("LIBRARIAN_BIN", binDir)
	got, err := InstallDir()
	if err != nil {
		t.Fatal(err)
	}
	if got != binDir {
		t.Errorf("InstallDir() = %q, want %q", got, binDir)
	}
}

func TestGetToolsEnv(t *testing.T) {
	binDir := t.TempDir()
	t.Setenv("LIBRARIAN_BIN", binDir)
	env, err := getToolsEnv()
	if err != nil {
		t.Fatal(err)
	}
	if got := env["PATH"]; got != binDir {
		t.Errorf("getToolsEnv()[PATH] = %q, want %q", got, binDir)
	}
}

func TestInstall_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		tools   *config.Tools
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
	} {
		t.Run(test.name, func(t *testing.T) {
			err := Install(t.Context(), test.tools)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Install() err = %v, wantErr = %v", err, test.wantErr)
			}
		})
	}
}
