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

package swift

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

func TestGeneratePackageSwift_WithDependencies(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "generated", "google-cloud-workflows-v1")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(filepath.Join(t.TempDir(), "generated"))

	service := &api.Service{Name: "Workflows", Package: "google.cloud.workflows.v1"}
	model := api.NewTestAPI(nil, nil, []*api.Service{service})
	model.PackageName = "google.cloud.workflows.v1"

	cfg := &parser.ModelConfig{
		Codec: map[string]string{
			"copyright-year": "2038",
		},
	}

	swiftCfg := &config.SwiftPackage{
		SwiftDefault: config.SwiftDefault{
			Dependencies: []config.SwiftDependency{
				{Name: "gax", Path: "packages/gax", RequiredByServices: true},
			},
		},
	}

	if err := Generate(t.Context(), model, outDir, cfg, swiftCfg); err != nil {
		t.Fatal(err)
	}

	packageSwiftPath := filepath.Join(outDir, "Package.swift")
	content, err := os.ReadFile(packageSwiftPath)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	gotPackageDeps := extractBlock(t, contentStr, "  dependencies: [", "\n  ],")
	wantPackageDeps := `  dependencies: [
    .package(path: "../../packages/gax"),
  ],`
	if diff := cmp.Diff(wantPackageDeps, gotPackageDeps); diff != "" {
		t.Errorf("mismatch in package dependencies (-want +got):\n%s", diff)
	}

	gotTargetDeps := extractBlock(t, contentStr, "      dependencies: [", "\n      ]")
	wantTargetDeps := `      dependencies: [
        .product(name: "gax", package: "gax"),
      ]`
	if diff := cmp.Diff(wantTargetDeps, gotTargetDeps); diff != "" {
		t.Errorf("mismatch in target dependencies (-want +got):\n%s", diff)
	}
}

func extractBlock(t *testing.T, content, startStr, endStr string) string {
	t.Helper()
	startIdx := strings.Index(content, startStr)
	if startIdx == -1 {
		t.Fatalf("Package.swift missing expected block start %q", startStr)
	}
	endIdx := strings.Index(content[startIdx:], endStr)
	if endIdx == -1 {
		t.Fatalf("Package.swift missing expected block end %q", endStr)
	}
	return content[startIdx : startIdx+endIdx+len(endStr)]
}
