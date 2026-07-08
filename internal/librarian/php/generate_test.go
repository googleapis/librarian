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

package php

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestGenerate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow integration test")
	}
	testhelper.RequireCommand(t, "php")
	testhelper.RequireCommand(t, "protoc")

	// Use mock googleapis checked in as test data
	googleapisDir := "../../testdata/googleapis"
	absGoogleapis, err := filepath.Abs(googleapisDir)
	if err != nil {
		t.Fatal(err)
	}

	repoRoot := t.TempDir()
	library := &config.Library{
		Name:   "secretmanager",
		Output: filepath.Join(repoRoot, "output"),
		APIs: []*config.API{
			{
				Path: "google/cloud/secretmanager/v1",
			},
		},
	}

	cfg := &config.Config{
		Language: config.LanguagePhp,
	}

	err = Generate(t.Context(), cfg, library, &sources.Sources{Googleapis: absGoogleapis})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify output
	outputDirs := []string{"src", "tests", "samples", "fragments"}
	for _, dir := range outputDirs {
		p := filepath.Join(library.Output, dir)
		if stat, err := os.Stat(p); err != nil || !stat.IsDir() {
			t.Errorf("expected directory %s to exist and be a directory", p)
		}
	}
}
