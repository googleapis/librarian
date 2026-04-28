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
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

func TestInstall_Error(t *testing.T) {
	if err := Install(t.Context(), nil); err == nil {
		t.Fatal("expected error when passing nil tools")
	}
}

func TestInstall_Success(t *testing.T) {
	gobin := t.TempDir()
	t.Setenv("GOBIN", gobin)
	
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
			path := filepath.Join(gobin, tool+suffix)
			if _, err := os.Stat(path); err != nil {
				t.Errorf("expected tool binary %s to exist: %v", tool, err)
			}
		})
	}
}
