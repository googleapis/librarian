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

package gcloud

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/sidekick/surfer/provider"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestGenerate_Parallelstore(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")

	googleapisDir := requireGoogleapisPath(t)
	scenarioInput, err := filepath.Abs("../../surfer/testdata/apis/parallelstore/input")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(scenarioInput, "parallelstore.proto")); err != nil {
		t.Skipf("parallelstore proto not found: %v", err)
	}

	protoRoot := filepath.Join(t.TempDir(), "proto_root")
	if err := os.MkdirAll(protoRoot, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(googleapisDir, "google"), filepath.Join(protoRoot, "google")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(scenarioInput, "parallelstore.proto"), filepath.Join(protoRoot, "parallelstore.proto")); err != nil {
		t.Fatal(err)
	}

	model, err := provider.CreateAPIModel(
		protoRoot,
		"parallelstore.proto",
		filepath.Join(scenarioInput, "service.yaml"),
		"",
		"",
	)
	if err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	if err := Generate(model, outDir, "example.com/parallelstore"); err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"go.mod",
		"main.go",
		"cmd/instances.go",
		"cmd/operations.go",
	} {
		t.Run("file_"+want, func(t *testing.T) {
			if _, err := os.Stat(filepath.Join(outDir, want)); err != nil {
				t.Errorf("expected file %q to exist: %v", want, err)
			}
		})
	}

	instancesPath := filepath.Join(outDir, "cmd", "instances.go")
	contents, err := os.ReadFile(instancesPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"instances"`,
		`"create"`,
		`"capacity-gib"`,
		`"description"`,
		`"deployment-type"`,
		`"import-data"`,
		`"export-data"`,
		`"list"`,
		`"describe"`,
		`"delete"`,
		`"update"`,
	} {
		t.Run("contains_"+want, func(t *testing.T) {
			if !strings.Contains(string(contents), want) {
				t.Errorf("instances.go does not contain %s", want)
			}
		})
	}

	t.Run("parses", func(t *testing.T) {
		fset := token.NewFileSet()
		for _, p := range []string{
			filepath.Join(outDir, "main.go"),
			filepath.Join(outDir, "cmd", "instances.go"),
			filepath.Join(outDir, "cmd", "operations.go"),
		} {
			if _, err := parser.ParseFile(fset, p, nil, parser.AllErrors); err != nil {
				t.Errorf("parser.ParseFile(%s): %v", p, err)
			}
		}
	})
}

func requireGoogleapisPath(t *testing.T) string {
	t.Helper()
	if env := os.Getenv("SURFER_GOOGLEAPIS"); env != "" {
		return env
	}
	candidates := []string{
		"../../../testdata/googleapis",
		"../../testdata/googleapis",
	}
	for _, rel := range candidates {
		if _, err := os.Stat(rel); err == nil {
			abs, err := filepath.Abs(rel)
			if err != nil {
				t.Fatal(err)
			}
			return abs
		}
	}
	t.Skip("core googleapis not found; set SURFER_GOOGLEAPIS or run from repo root")
	return ""
}
