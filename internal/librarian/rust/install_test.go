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

package rust

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

func TestInstall(t *testing.T) {
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "cargo"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))

	tools := &config.Tools{
		Cargo: []*config.CargoTool{
			{Name: "cargo-semver-checks", Version: "0.46.0"},
			{Name: "taplo-cli", Version: "0.10.0"},
		},
	}
	if err := Install(t.Context(), tools); err != nil {
		t.Fatal(err)
	}
}

func TestInstall_MissingVersion(t *testing.T) {
	tools := &config.Tools{
		Cargo: []*config.CargoTool{
			{Name: "some-tool"},
		},
	}
	err := Install(t.Context(), tools)
	if !errors.Is(err, ErrMissingToolVersion) {
		t.Fatalf("got %v, want %v", err, ErrMissingToolVersion)
	}
}

func TestInstall_NoCargoTools(t *testing.T) {
	for _, test := range []struct {
		name  string
		tools *config.Tools
	}{
		{
			name: "nil tools",
		},
		{
			name:  "no Cargo tools",
			tools: &config.Tools{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := Install(t.Context(), &config.Tools{}); err != nil {
				t.Fatal(err)
			}
		})
	}
}
