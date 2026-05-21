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

package pip

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestInstall(t *testing.T) {
	tmpDir := t.TempDir()
	stubLogPath := filepath.Join(tmpDir, "pip_invocations.log")
	stubContent := fmt.Sprintf(`#!/bin/sh
echo "pip $@" >> %q
`, stubLogPath)
	stubDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(stubDir, 0755); err != nil {
		t.Fatal(err)
	}
	stubPath := filepath.Join(stubDir, "pip")
	if err := os.WriteFile(stubPath, []byte(stubContent), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", stubDir)
	localPkgPath := filepath.Join(tmpDir, "mylocalpkg")
	if err := os.MkdirAll(localPkgPath, 0755); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name     string
		tools    []*config.PipTool
		wantArgs string
	}{
		{
			name: "install external packages",
			tools: []*config.PipTool{
				{Name: "PyYAML", Version: "6.0.2"},
				{Name: "jinja2", Version: "3.1.6"},
			},
			wantArgs: "install PyYAML==6.0.2 jinja2==3.1.6",
		},
		{
			name: "install external packages with raw package spec",
			tools: []*config.PipTool{
				{Name: "synthtool", Package: "git+https://github.com/..."},
			},
			wantArgs: "install git+https://github.com/...",
		},
		{
			name: "install package with name only (no version/package)",
			tools: []*config.PipTool{
				{Name: "requests"},
			},
			wantArgs: "install requests",
		},
		{
			name: "install local package path",
			tools: []*config.PipTool{
				{Name: "synthtool", LocalPath: localPkgPath},
			},
			wantArgs: "install " + localPkgPath,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_ = os.Remove(stubLogPath)
			err := Install(t.Context(), test.tools)
			if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(stubLogPath)
			if err != nil {
				t.Fatal(err)
			}
			gotInvocations := strings.TrimSpace(string(data))
			wantInvocation := "pip " + test.wantArgs
			if diff := cmp.Diff(wantInvocation, gotInvocations); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInstall_Error(t *testing.T) {
	tmpDir := t.TempDir()
	stubDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(stubDir, 0755); err != nil {
		t.Fatal(err)
	}
	stubPath := filepath.Join(stubDir, "pip")
	if err := os.WriteFile(stubPath, []byte("#!/bin/sh\nexit 1\n"), 0755); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name    string
		tools   []*config.PipTool
		setup   func(t *testing.T)
		wantErr error
	}{
		{
			name: "pip command fails",
			tools: []*config.PipTool{
				{Name: "failpkg"},
			},
			setup: func(t *testing.T) {
				t.Setenv("PATH", stubDir)
			},
			wantErr: ErrInstall,
		},
		{
			name: "local path not found",
			tools: []*config.PipTool{
				{Name: "failpkg", LocalPath: filepath.Join(tmpDir, "nonexistentpkg")},
			},
			wantErr: ErrLocalPathNotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.setup != nil {
				test.setup(t)
			}
			err := Install(t.Context(), test.tools)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("Install() error = %v, wantErr = %v", err, test.wantErr)
			}
		})
	}
}
