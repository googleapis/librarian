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
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/cache"
	"github.com/googleapis/librarian/internal/config"
)

func TestMergeEnv(t *testing.T) {
	for _, test := range []struct {
		name string
		env  map[string]string
		path string
		want func(base string) map[string]string
	}{
		{
			name: "nil env",
			env:  nil,
			path: "/original/path",
			want: func(base string) map[string]string {
				return map[string]string{
					envPath: filepath.Join(base, toolsDir) + ":/original/path",
				}
			},
		},
		{
			name: "custom env keys merged",
			env: map[string]string{
				"FOO": "bar",
			},
			path: "/original/path",
			want: func(base string) map[string]string {
				return map[string]string{
					envPath: filepath.Join(base, toolsDir) + ":/original/path",
					"FOO":   "bar",
				}
			},
		},
		{
			name: "env overrides PATH",
			env: map[string]string{
				envPath: "/env/custom/path",
			},
			path: "/original/path",
			want: func(base string) map[string]string {
				return map[string]string{
					envPath: "/env/custom/path",
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			baseDir := t.TempDir()
			t.Setenv(cache.EnvLibrarianBin, baseDir)
			t.Setenv(envPath, test.path)
			got, err := mergeEnv(test.env)
			if err != nil {
				t.Fatal(err)
			}
			wantAbsBase, err := filepath.Abs(baseDir)
			if err != nil {
				t.Fatal(err)
			}
			wantMap := test.want(wantAbsBase)
			if diff := cmp.Diff(wantMap, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRunProtoc(t *testing.T) {
	stubName := "protoc"
	if runtime.GOOS == "windows" {
		stubName += ".exe"
	}
	for _, test := range []struct {
		name  string
		setup func(t *testing.T) *config.Protoc
	}{
		{
			name: "nil config uses system protoc in PATH",
			setup: func(t *testing.T) *config.Protoc {
				stubDir := t.TempDir()
				createStubExecutable(t, filepath.Join(stubDir, stubName))
				t.Setenv(envPath, stubDir+string(filepath.ListSeparator)+os.Getenv(envPath))
				return nil
			},
		},
		{
			name: "configured protoc uses installed tool",
			setup: func(t *testing.T) *config.Protoc {
				binDir := t.TempDir()
				t.Setenv(cache.EnvLibrarianBin, binDir)
				version := "25.1"
				protocPath := filepath.Join(binDir, "protoc", "v"+version, "bin", stubName)
				createStubExecutable(t, protocPath)
				return &config.Protoc{Version: version}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			pc := test.setup(t)
			if err := runProtoc(t.Context(), pc, "--version"); err != nil {
				t.Fatalf("runProtoc() error = %v, want nil", err)
			}
		})
	}
}

func createStubExecutable(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	content := "#!/bin/sh\nexit 0\n"
	if runtime.GOOS == "windows" {
		content = "@echo off\r\nexit /b 0\r\n"
	}
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
}
