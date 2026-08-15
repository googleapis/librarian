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

package swift

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/cache"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestInstall_NilOrEmptyTools(t *testing.T) {
	for _, test := range []struct {
		name  string
		tools *config.Tools
	}{
		{
			name:  "nil tools",
			tools: nil,
		},
		{
			name:  "empty tools",
			tools: &config.Tools{},
		},
		{
			name: "with protoc only",
			tools: &config.Tools{
				Protoc: &config.Protoc{Version: "29.3"},
			},
		},
		{
			name: "empty swift tools slice",
			tools: &config.Tools{
				Swift: []*config.SwiftTool{},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := Install(t.Context(), test.tools); err != nil {
				t.Fatalf("Install() unexpected error = %v", err)
			}
		})
	}
}

func TestInstall_MissingSwift(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows, bash script set up does not work")
	}
	bin := t.TempDir()
	testhelper.WriteExecutable(t, filepath.Join(bin, "swift-format"), "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", bin)

	tools := &config.Tools{
		Swift: []*config.SwiftTool{
			{Name: "protoc-gen-swift", Repo: "https://github.com/apple/swift-protobuf.git", Version: "1.38.1"},
		},
	}
	err := Install(t.Context(), tools)
	if err == nil || !errors.Is(err, errMissingExecutable) || !strings.Contains(err.Error(), "swift") {
		t.Fatalf("got %v, want error containing swift and %v", err, errMissingExecutable)
	}
}

func TestInstall_MissingSwiftFormat(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows, bash script set up does not work")
	}
	bin := t.TempDir()
	testhelper.WriteExecutable(t, filepath.Join(bin, "swift"), "#!/bin/sh\necho 'Swift version 6.2'\n")
	t.Setenv("PATH", bin)

	tools := &config.Tools{
		Swift: []*config.SwiftTool{
			{Name: "protoc-gen-swift", Repo: "https://github.com/apple/swift-protobuf.git", Version: "1.38.1"},
		},
	}
	err := Install(t.Context(), tools)
	if err == nil || !errors.Is(err, errMissingExecutable) || !strings.Contains(err.Error(), "swift-format") {
		t.Fatalf("got %v, want error containing swift-format and %v", err, errMissingExecutable)
	}
}

func TestInstall_MissingGit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows, bash script set up does not work")
	}
	bin := t.TempDir()
	testhelper.WriteExecutable(t, filepath.Join(bin, "swift"), "#!/bin/sh\necho 'Swift version 6.2'\n")
	testhelper.WriteExecutable(t, filepath.Join(bin, "swift-format"), "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", bin)

	tools := &config.Tools{
		Swift: []*config.SwiftTool{
			{Name: "protoc-gen-swift", Repo: "https://github.com/apple/swift-protobuf.git", Version: "1.38.1"},
		},
	}
	err := Install(t.Context(), tools)
	if err == nil || !errors.Is(err, errMissingExecutable) || !strings.Contains(err.Error(), "git") {
		t.Fatalf("got %v, want error containing git and %v", err, errMissingExecutable)
	}
}

func TestCheckSwiftVersionOutput(t *testing.T) {
	for _, test := range []struct {
		name    string
		output  string
		wantErr error
	}{
		{
			name:   "Apple Swift 6.2.4",
			output: "Apple Swift version 6.2.4 (swift-6.2.4-RELEASE)\nTarget: arm64-apple-macosx26.0\n",
		},
		{
			name:   "Apple Swift 6.2 release",
			output: "Apple Swift version 6.2 (swiftlang-6.2.0.16.20 clang-1600.0.30.6)\n",
		},
		{
			name:   "Linux Swift 6.2",
			output: "Swift version 6.2 (swift-6.2-RELEASE)\n",
		},
		{
			name:   "Swift 6.3 dev",
			output: "Swift version 6.3-dev\n",
		},
		{
			name:   "Swift 7.0",
			output: "Swift version 7.0\n",
		},
		{
			name:    "Swift 6.1 too low",
			output:  "Apple Swift version 6.1.3 (swiftlang-6.1.3)\n",
			wantErr: errSwiftVersionTooLow,
		},
		{
			name:    "Swift 5.10 too low",
			output:  "Swift version 5.10.1\n",
			wantErr: errSwiftVersionTooLow,
		},
		{
			name:    "Invalid output",
			output:  "command not found\n",
			wantErr: errCannotParseSwiftVersion,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := checkSwiftVersionOutput(test.output)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("checkSwiftVersionOutput() error = %v, want %v", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("checkSwiftVersionOutput() unexpected error = %v", err)
			}
		})
	}
}

func TestInstall_InvalidToolConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows, bash script set up does not work")
	}
	bin := t.TempDir()
	testhelper.WriteExecutable(t, filepath.Join(bin, "swift"), "#!/bin/sh\necho 'Swift version 6.2'\n")
	testhelper.WriteExecutable(t, filepath.Join(bin, "swift-format"), "#!/bin/sh\nexit 0\n")
	testhelper.WriteExecutable(t, filepath.Join(bin, "git"), "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", bin)

	for _, test := range []struct {
		name string
		tool *config.SwiftTool
	}{
		{
			name: "empty name",
			tool: &config.SwiftTool{
				Repo:    "https://github.com/apple/swift-protobuf.git",
				Version: "1.38.1",
			},
		},
		{
			name: "both local_path and repo",
			tool: &config.SwiftTool{
				Name:      "tool",
				LocalPath: "/tmp/local",
				Repo:      "https://github.com/apple/swift-protobuf.git",
				Version:   "1.38.1",
			},
		},
		{
			name: "neither local_path nor repo",
			tool: &config.SwiftTool{
				Name: "tool",
			},
		},
		{
			name: "repo without version",
			tool: &config.SwiftTool{
				Name: "tool",
				Repo: "https://github.com/apple/swift-protobuf.git",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tools := &config.Tools{
				Swift: []*config.SwiftTool{test.tool},
			}
			err := Install(t.Context(), tools)
			if !errors.Is(err, errInvalidTool) {
				t.Fatalf("got %v, want %v", err, errInvalidTool)
			}
		})
	}
}

func TestInstall_LocalPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows, bash script set up does not work")
	}
	localPkg := t.TempDir()
	bin := t.TempDir()
	testhelper.WriteExecutable(t, filepath.Join(bin, "swift-format"), "#!/bin/sh\nexit 0\n")
	testhelper.WriteExecutable(t, filepath.Join(bin, "git"), "#!/bin/sh\nexit 0\n")

	swiftScript := `#!/bin/sh
if [ "$1" = "--version" ]; then
    echo "Swift version 6.2"
    exit 0
fi
if [ "$1" = "build" ]; then
    mkdir -p .build/release
    printf '#!/bin/sh\necho "built-tool"\n' > .build/release/local-tool
    chmod +x .build/release/local-tool
		printf "${PWD}/.build/release"
    exit 0
fi
exit 1
`
	testhelper.WriteExecutable(t, filepath.Join(bin, "swift"), swiftScript)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	customBinDir := t.TempDir()
	t.Setenv(cache.EnvLibrarianBin, customBinDir)

	tools := &config.Tools{
		Swift: []*config.SwiftTool{
			{Name: "local-tool", LocalPath: localPkg},
		},
	}
	if err := Install(t.Context(), tools); err != nil {
		t.Fatalf("Install() unexpected error = %v", err)
	}

	installedBinary := filepath.Join(customBinDir, toolsDir, "bin", "local-tool")
	if _, err := os.Stat(installedBinary); err != nil {
		t.Fatalf("expected binary at %s not found: %v", installedBinary, err)
	}
}

func TestInstall_RemoteRepo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows, bash script set up does not work")
	}
	bin := t.TempDir()
	testhelper.WriteExecutable(t, filepath.Join(bin, "swift-format"), "#!/bin/sh\nexit 0\n")

	gitScript := `#!/bin/sh
if [ "$1" = "clone" ]; then
    mkdir -p "$7"
    exit 0
fi
exit 0
`
	testhelper.WriteExecutable(t, filepath.Join(bin, "git"), gitScript)

	swiftScript := `#!/bin/sh
if [ "$1" = "--version" ]; then
    echo "Swift version 6.2"
    exit 0
fi
if [ "$1" = "build" ]; then
    mkdir -p .build/release
    printf '#!/bin/sh\necho "grpc-plugin"\n' > .build/release/protoc-gen-grpc-swift
    chmod +x .build/release/protoc-gen-grpc-swift
		printf "${PWD}/.build/release"
    exit 0
fi
exit 1
`
	testhelper.WriteExecutable(t, filepath.Join(bin, "swift"), swiftScript)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	customBinDir := t.TempDir()
	t.Setenv(cache.EnvLibrarianBin, customBinDir)

	tools := &config.Tools{
		Swift: []*config.SwiftTool{
			{
				Name:    "protoc-gen-grpc-swift",
				Repo:    "grpc/grpc-swift",
				Version: "1.23.0",
				Product: "protoc-gen-grpc-swift",
			},
		},
	}
	if err := Install(t.Context(), tools); err != nil {
		t.Fatalf("Install() unexpected error = %v", err)
	}

	installedBinary := filepath.Join(customBinDir, toolsDir, "bin", "protoc-gen-grpc-swift")
	if _, err := os.Stat(installedBinary); err != nil {
		t.Fatalf("expected binary at %s not found: %v", installedBinary, err)
	}
}

func TestInstallDir_And_BinDir(t *testing.T) {
	for _, test := range []struct {
		name    string
		env     map[string]string
		wantDir string
		wantBin string
	}{
		{
			name:    "LIBRARIAN_BIN set",
			env:     map[string]string{cache.EnvLibrarianBin: "/custom/bin"},
			wantDir: "/custom/bin/swift_tools",
			wantBin: "/custom/bin/swift_tools/bin",
		},
		{
			name:    "LIBRARIAN_CACHE set",
			env:     map[string]string{cache.EnvLibrarianCache: "/custom/cache", cache.EnvLibrarianBin: ""},
			wantDir: "/custom/cache/bin/swift_tools",
			wantBin: "/custom/cache/bin/swift_tools/bin",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			for k, v := range test.env {
				t.Setenv(k, v)
			}
			gotDir, err := InstallDir()
			if err != nil {
				t.Fatalf("InstallDir() error = %v", err)
			}
			if gotDir != test.wantDir {
				t.Errorf("InstallDir() = %q, want %q", gotDir, test.wantDir)
			}
			gotBin, err := binDir()
			if err != nil {
				t.Fatalf("binDir() error = %v", err)
			}
			if gotBin != test.wantBin {
				t.Errorf("binDir() = %q, want %q", gotBin, test.wantBin)
			}
			env, err := toolsEnv()
			if err != nil {
				t.Fatalf("toolsEnv() error = %v", err)
			}
			if env["PATH"] != test.wantBin {
				t.Errorf("toolsEnv()[PATH] = %q, want %q", env["PATH"], test.wantBin)
			}
		})
	}
}
