// Copyright 2025 Google LLC
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
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/cache"
	"github.com/googleapis/librarian/internal/config"
)

func TestInstall_Error(t *testing.T) {
	for _, test := range []struct {
		name  string
		tools *config.Tools
	}{
		{"nil tools", nil},
		{"empty tools", &config.Tools{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := Install(t.Context(), test.tools); !errors.Is(err, errNoToolsSpecified) {
				t.Fatalf("Install() error = %v, want %v", err, errNoToolsSpecified)
			}
		})
	}
}

func TestInstall_Success(t *testing.T) {
	installDir := t.TempDir()
	t.Setenv(cache.EnvLibrarianBin, installDir)
	tools := &config.Tools{
		Go: []*config.GoTool{
			{Name: "github.com/googleapis/gapic-generator-go/cmd/protoc-gen-go_gapic", Version: "v0.58.0"},
			{Name: "golang.org/x/tools/cmd/goimports", Version: "v0.44.0"},
			{Name: "google.golang.org/grpc/cmd/protoc-gen-go-grpc", Version: "v1.3.0"},
			{Name: "google.golang.org/protobuf/cmd/protoc-gen-go", Version: "v1.36.11"},
		},
	}
	if err := Install(t.Context(), tools); err != nil {
		t.Fatal(err)
	}
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	for _, tool := range []string{
		"protoc-gen-go_gapic",
		"goimports",
		"protoc-gen-go-grpc",
		"protoc-gen-go",
	} {
		t.Run(tool, func(t *testing.T) {
			path := filepath.Join(installDir, toolsDir, tool+suffix)
			if _, err := os.Stat(path); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestGetInstallDir(t *testing.T) {
	for _, test := range []struct {
		name string
		env  map[string]string
		want string
	}{
		{
			name: "LIBRARIAN_BIN set",
			env:  map[string]string{cache.EnvLibrarianBin: "/custom/install/dir"},
			want: "/custom/install/dir/go_tools",
		},
		{
			name: "LIBRARIAN_BIN empty",
			env:  map[string]string{cache.EnvLibrarianCache: "/my/home/cache"},
			want: "/my/home/cache/bin/go_tools",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			for k, v := range test.env {
				t.Setenv(k, v)
			}
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
