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
)

func TestInstall(t *testing.T) {
	gobin := t.TempDir()
	t.Setenv("GOBIN", gobin)
	if err := Install(t.Context(), nil); err != nil {
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
