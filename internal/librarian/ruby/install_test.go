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
	"os"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

func TestInstall(t *testing.T) {
	stubDir := t.TempDir()
	gemStubPath := filepath.Join(stubDir, "gem")
	recordFile := filepath.Join(t.TempDir(), "calls.txt")
	stubContent := "#!/bin/sh\necho \"$*\" >> \"" + recordFile + "\"\nexit 0\n"
	if err := os.WriteFile(gemStubPath, []byte(stubContent), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", stubDir+string(filepath.ListSeparator)+os.Getenv("PATH"))
	tools := &config.Tools{
		Gem: []*config.GemTool{
			{
				Name:    "gapic-generator",
				Version: "1.2.3",
			},
		},
	}
	if err := Install(t.Context(), tools); err != nil {
		t.Fatalf("Install() returned unexpected error: %v", err)
	}
	data, err := os.ReadFile(recordFile)
	if err != nil {
		t.Fatalf("failed to read call records: %v", err)
	}
	got := string(data)
	want := "install gapic-generator -v 1.2.3 --no-document\n"
	if got != want {
		t.Errorf("gem called with = %q, want %q", got, want)
	}
}

func TestInstallDir(t *testing.T) {
	binDir := t.TempDir()
	t.Setenv("LIBRARIAN_BIN", binDir)
	got, err := InstallDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(binDir, "ruby_tools")
	if got != want {
		t.Errorf("InstallDir() = %q, want %q", got, want)
	}
}

func TestBinDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LIBRARIAN_BIN", dir)
	got, err := binDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "ruby_tools", "bin")
	if got != want {
		t.Errorf("binDir() = %q, want %q", got, want)
	}
}

func TestVerify(t *testing.T) {
	stubDir := t.TempDir()
	gemStubPath := filepath.Join(stubDir, "gem")
	// Create a simple shell script stub for "gem".
	stubContent := "#!/bin/sh\nexit 0\n"
	if err := os.WriteFile(gemStubPath, []byte(stubContent), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", stubDir+string(filepath.ListSeparator)+os.Getenv("PATH"))
	tools := &config.Tools{
		Gem: []*config.GemTool{
			{
				Name:    "a-gem-tool",
				Version: "1.0",
			},
		},
	}

	if err := verify(tools); err != nil {
		t.Errorf("verify() returned unexpected error: %v", err)
	}
}

func TestVerify_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		tools   *config.Tools
		setup   func(t *testing.T)
		wantErr error
	}{
		{
			name:    "nil tools",
			tools:   nil,
			wantErr: errNoGems,
		},
		{
			name:    "empty tools",
			tools:   &config.Tools{},
			wantErr: errNoGems,
		},
		{
			name: "missing gem in path",
			tools: &config.Tools{
				Gem: []*config.GemTool{
					{
						Name:    "a-gem-tool",
						Version: "1.0",
					},
				},
			},
			setup: func(t *testing.T) {
				t.Setenv("PATH", t.TempDir())
			},
			wantErr: errMissingExecutable,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.setup != nil {
				test.setup(t)
			}
			err := verify(test.tools)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("verify() error = %v, wantErr = %v", err, test.wantErr)
			}
		})
	}
}
