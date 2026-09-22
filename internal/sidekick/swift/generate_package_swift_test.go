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
)

func TestGeneratePackageSwift_WithDependencies(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	outDir := filepath.Join("generated", "google-cloud-workflows-v1")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll("generated")

	service := &api.Service{Name: "Workflows", Package: "google.cloud.workflows.v1"}
	model := api.NewTestAPI(nil, nil, []*api.Service{service})
	model.PackageName = "google.cloud.workflows.v1"

	swiftCfg := &config.SwiftPackage{
		SwiftDefault: config.SwiftDefault{
			Dependencies: []config.SwiftDependency{
				{Name: "auth", URL: "https://github.com/googleapis/swift-google-auth", Path: "pkgs/swift-google-auth", Version: "0.2.0", RequiredByServices: true},
				{Name: "gax", Path: "packages/gax", RequiredByServices: true},
				{Name: "wkt", ApiPackage: "google.protobuf", Path: "packages/wkt"},
				{Name: "proto", URL: "https://github.com/apple/swift-protobuf", Version: "1.36.1", RequiredByServices: true},
			},
		},
	}

	library := &config.Library{
		Swift: swiftCfg,
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	packageSwiftPath := filepath.Join(outDir, "Package.swift")
	content, err := os.ReadFile(packageSwiftPath)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	gotPackage := extractBlock(t, contentStr, "let package = Package(", "\n  platforms:")
	wantPackage := `let package = Package(
  name: "GoogleCloudWorkflowsV1",
  platforms:`
	if diff := cmp.Diff(wantPackage, gotPackage); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	gotPackageDeps := extractBlock(t, contentStr, "  dependencies: [", "\n  ],")
	wantPackageDeps := `  dependencies: [
    localOrRemotePackage(
      url: "https://github.com/googleapis/swift-google-auth",
      path: "pkgs/swift-google-auth",
      from: "0.2.0"
    ),
    .package(path: "../../packages/gax"),
    .package(url: "https://github.com/apple/swift-protobuf", from: "1.36.1"),
    .package(path: "../../packages/wkt"),
  ],`
	if diff := cmp.Diff(wantPackageDeps, gotPackageDeps); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	gotTargetDeps := extractBlock(t, contentStr, "      dependencies: [", "\n      ]")
	wantTargetDeps := `      dependencies: [
        .product(name: "auth", package: "swift-google-auth"),
        .product(name: "gax", package: "gax"),
        .product(name: "proto", package: "swift-protobuf"),
        .product(name: "wkt", package: "wkt"),
      ]`
	if diff := cmp.Diff(wantTargetDeps, gotTargetDeps); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	gotSwiftSettings := extractBlock(t, contentStr, "      swiftSettings: [", "\n      ]")
	wantSwiftSettings := `      swiftSettings: [
        .enableUpcomingFeature("InternalImportsByDefault"),
      ]`
	if diff := cmp.Diff(wantSwiftSettings, gotSwiftSettings); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	gotHelper := extractBlock(t, contentStr, "func localOrRemotePackage(", "  return .package(url: url, from: version)\n}\n")
	wantHelper := `func localOrRemotePackage(url: String, path: String, from version: Version) -> Package.Dependency {
  if let env = Context.environment["GOOGLE_CLOUD_SWIFT_LOCAL_DEPS"], !env.isEmpty {
    let root = (env == "1" || env == "true") ? "\(Context.packageDirectory)/../.." : env
    return .package(path: "\(root)/\(path)")
  }
  return .package(url: url, from: version)
}
`
	if diff := cmp.Diff(wantHelper, gotHelper); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func extractBlock(t *testing.T, content, startStr, endStr string) string {
	t.Helper()
	startIdx := strings.Index(content, startStr)
	if startIdx == -1 {
		t.Fatalf("missing expected block start %q\n\n%s", startStr, content)
	}
	endIdx := strings.Index(content[startIdx:], endStr)
	if endIdx == -1 {
		t.Fatalf("missing expected block end %q\n\n%s", endStr, content)
	}
	return content[startIdx : startIdx+endIdx+len(endStr)]
}
