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

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallPipTools(t *testing.T) {
	// 1. Create a temporary directory for the stub executable and test inputs
	tmpDir := t.TempDir()

	// 2. Create a stub "pip" script that writes its arguments to a log file
	stubLogPath := filepath.Join(tmpDir, "pip_invocations.log")
	stubContent := fmt.Sprintf(`#!/bin/bash
if [[ "$*" == *"failpkg"* ]]; then
  exit 1
fi
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

	// 3. Set PATH so our stub "pip" is executed instead of system pip
	t.Setenv("PATH", stubDir)

	// 4. Create some local packages we want to test
	localPkgDir := filepath.Join(tmpDir, "my_local_pkg")
	if err := os.MkdirAll(localPkgDir, 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		tools       []*PipTool
		wantArgs    string
		expectError bool
	}{
		{
			name: "install external packages",
			tools: []*PipTool{
				{Name: "PyYAML", Version: "6.0.2"},
				{Name: "jinja2", Version: "3.1.6"},
			},
			wantArgs: "install PyYAML==6.0.2 jinja2==3.1.6",
		},
		{
			name: "install external packages with raw package spec",
			tools: []*PipTool{
				{Name: "synthtool", Package: "git+https://github.com/..."},
			},
			wantArgs: "install git+https://github.com/...",
		},
		{
			name: "install local package",
			tools: []*PipTool{
				{Name: "mylocal", LocalPath: localPkgDir},
			},
			wantArgs: "install --no-build-isolation " + localPkgDir,
		},
		{
			name: "install mixed local and external",
			tools: []*PipTool{
				{Name: "mylocal", LocalPath: localPkgDir},
				{Name: "PyYAML", Version: "6.0.2"},
			},
			wantArgs: "install --no-build-isolation " + localPkgDir + " PyYAML==6.0.2",
		},
		{
			name: "install local package missing error",
			tools: []*PipTool{
				{Name: "mylocal", LocalPath: filepath.Join(tmpDir, "does_not_exist")},
			},
			expectError: true,
		},
		{
			name: "install package with name only (no version/package/local_path)",
			tools: []*PipTool{
				{Name: "requests"},
			},
			wantArgs: "install requests",
		},
		{
			name: "pip command execution failure",
			tools: []*PipTool{
				{Name: "failpkg"},
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Clear the log file before each test
			_ = os.Remove(stubLogPath)

			err := InstallPipTools(t.Context(), tc.tools)
			if tc.expectError {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			// Verify the stub log file contains the expected arguments
			data, err := os.ReadFile(stubLogPath)
			if err != nil {
				t.Fatal(err)
			}
			gotInvocations := strings.TrimSpace(string(data))
			wantInvocation := "pip " + tc.wantArgs
			if gotInvocations != wantInvocation {
				t.Errorf("unexpected invocations:\n got: %q\nwant: %q", gotInvocations, wantInvocation)
			}
		})
	}
}

func TestInstallPipTools_AbsPathError(t *testing.T) {
	tmpDir := t.TempDir()

	targetDir := filepath.Join(tmpDir, "delete_me")
	if err := os.Mkdir(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(targetDir)
	if err := os.RemoveAll(targetDir); err != nil {
		t.Fatal(err)
	}

	tools := []*PipTool{
		{Name: "mylocal", LocalPath: "relative_path_will_fail"},
	}
	err := InstallPipTools(t.Context(), tools)
	if err == nil {
		t.Errorf("expected error due to deleted working directory but got nil")
	}
}
