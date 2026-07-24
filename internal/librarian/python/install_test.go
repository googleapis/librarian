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

package python

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

func TestInstall(t *testing.T) {
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "pip"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	t.Run("fallback to embedded librarian.yaml", func(t *testing.T) {
		if err := Install(t.Context(), nil); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("use tools from config", func(t *testing.T) {
		tools := &config.Tools{
			Pip: []*config.PipTool{
				{Name: "ruff", Version: "0.14.14"},
			},
		}
		if err := Install(t.Context(), tools); err != nil {
			t.Fatal(err)
		}
	})
}

