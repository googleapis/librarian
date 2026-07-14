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

package golang

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/cache"
	"github.com/googleapis/librarian/internal/config"
)

func TestRunProtoc(t *testing.T) {
	stubName := "protoc"
	if runtime.GOOS == "windows" {
		stubName += ".exe"
	}
	for _, test := range []struct {
		name  string
		setup func(t *testing.T, recordFile string) (*config.Protoc, string)
	}{
		{
			name: "nil config uses system protoc in PATH",
			setup: func(t *testing.T, recordFile string) (*config.Protoc, string) {
				stubDir := t.TempDir()
				path := filepath.Join(stubDir, stubName)
				createStubExecutable(t, path, recordFile)
				t.Setenv(envPath, stubDir+string(filepath.ListSeparator)+os.Getenv(envPath))
				return nil, path
			},
		},
		{
			name: "configured protoc uses installed tool",
			setup: func(t *testing.T, recordFile string) (*config.Protoc, string) {
				binDir := t.TempDir()
				t.Setenv(cache.EnvLibrarianBin, binDir)
				version := "25.1"
				protocPath := filepath.Join(binDir, "protoc", "v"+version, "bin", stubName)
				createStubExecutable(t, protocPath, recordFile)
				return &config.Protoc{Version: version}, protocPath
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			recordFile := filepath.Join(t.TempDir(), "calls.txt")
			pc, want := test.setup(t, recordFile)
			if err := runProtoc(t.Context(), pc, "--version"); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(recordFile)
			if err != nil {
				t.Fatal(err)
			}
			got := strings.TrimSpace(string(data))
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInstallEnv(t *testing.T) {
	for _, test := range []struct {
		name   string
		binDir func(t *testing.T) string
		want   func(absBin string) map[string]string
	}{
		{
			name: "returns PATH pointing to go_tools directory",
			binDir: func(t *testing.T) string {
				return t.TempDir()
			},
			want: func(absBin string) map[string]string {
				return map[string]string{
					envPath: filepath.Join(absBin, toolsDir),
				}
			},
		},
		{
			name: "handles nested bin directory path",
			binDir: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nested", "bin")
			},
			want: func(absBin string) map[string]string {
				return map[string]string{
					envPath: filepath.Join(absBin, toolsDir),
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			binDir := test.binDir(t)
			t.Setenv(cache.EnvLibrarianBin, binDir)
			got, err := installEnv()
			if err != nil {
				t.Fatal(err)
			}
			wantAbsBin, err := filepath.Abs(binDir)
			if err != nil {
				t.Fatal(err)
			}
			wantMap := test.want(wantAbsBin)
			if diff := cmp.Diff(wantMap, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInstallEnv_Error(t *testing.T) {
	t.Setenv(cache.EnvLibrarianBin, "")
	t.Setenv(cache.EnvLibrarianCache, "")
	t.Setenv("HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")
	if _, err := installEnv(); err == nil {
		t.Error("installEnv() error = nil, want error")
	}
}

func createStubExecutable(t *testing.T, path, recordFile string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	content := fmt.Sprintf("#!/bin/sh\necho \"$0\" >> %q\nexit 0\n", recordFile)
	if runtime.GOOS == "windows" {
		content = fmt.Sprintf("@echo off\r\necho %%0 >> %q\r\nexit /b 0\r\n", recordFile)
	}
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
}
