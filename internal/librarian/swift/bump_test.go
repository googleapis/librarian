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
	"bytes"
	"os"
	"path"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

const (
	testLibraryName = "GoogleCloudSecretManagerV1"
	testPackageName = "google-cloud-secretmanager-v1"
)

func testManifest() string {
	return path.Join(testPackageName, "Sources", testLibraryName, clientsManifest)
}

func TestVersionAlreadyBumpedSuccess(t *testing.T) {
	const tag = "package-version-update-success"
	testhelper.RequireCommand(t, "git")
	setupForSwiftVersionBump(t, tag)

	name := testManifest()
	contents, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	contents = bytes.ReplaceAll(contents, []byte("1.0.0"), []byte("2.3.4"))
	if err := os.WriteFile(name, contents, 0o644); err != nil {
		t.Fatal(err)
	}
	testhelper.RunGit(t, "commit", "-m", "updated version", ".")

	bumped, err := versionAlreadyBumped(t.Context(), "git", tag, name)
	if err != nil {
		t.Fatal(err)
	}
	if !bumped {
		t.Errorf("expected versionAlreadyBumped() == true, got false")
	}
}

func TestVersionAlreadyBumpedNewPackage(t *testing.T) {
	const tag = "package-version-update-new-package"
	testhelper.RequireCommand(t, "git")
	setupForSwiftVersionBump(t, tag)

	testhelper.AddSwiftPackage(t, "google-cloud-new", "GoogleCloudNew")
	testhelper.RunGit(t, "add", ".")
	testhelper.RunGit(t, "commit", "-m", "new package", ".")

	name := path.Join("google-cloud-new", "Sources", "GoogleCloudNew", clientsManifest)
	bumped, err := versionAlreadyBumped(t.Context(), "git", tag, name)
	if err != nil {
		t.Fatal(err)
	}
	if bumped {
		t.Errorf("expected versionAlreadyBumped() == false on a new package, got true")
	}
}

func TestVersionAlreadyBumpedNoChange(t *testing.T) {
	const tag = "package-version-update-no-change"
	testhelper.RequireCommand(t, "git")
	setupForSwiftVersionBump(t, tag)
	name := testManifest()
	bumped, err := versionAlreadyBumped(t.Context(), "git", tag, name)
	if err != nil {
		t.Fatal(err)
	}
	if bumped {
		t.Errorf("expected versionAlreadyBumped() == false, got true")
	}
}

func TestVersionAlreadyBumpedBadDiff(t *testing.T) {
	const tag = "package-version-update-success"
	testhelper.RequireCommand(t, "git")
	setupForSwiftVersionBump(t, tag)
	name := testManifest()
	if updated, err := versionAlreadyBumped(t.Context(), "git", "not-a-valid-tag", name); err == nil {
		t.Errorf("expected an error with an invalid tag, got=%v", updated)
	}
}

func TestVersionBadDirectory(t *testing.T) {
	const tag = "package-version-update-success"
	testhelper.RequireCommand(t, "git")
	setupForSwiftVersionBump(t, tag)
	name := path.Join("not-the-right-package", "Sources", "NotTheRightLibrary", clientsManifest)
	if updated, err := versionAlreadyBumped(t.Context(), "git", "not-a-valid-tag", name); err == nil {
		t.Errorf("expected an error with an invalid tag, got=%v", updated)
	}
}

func setupForSwiftVersionBump(t *testing.T, wantTag string) {
	remoteDir := t.TempDir()
	testhelper.ContinueInNewGitRepository(t, remoteDir)
	testhelper.AddSwiftPackage(t, testPackageName, testLibraryName)
	testhelper.RunGit(t, "add", ".")
	testhelper.RunGit(t, "commit", "-m", "initial version")
	testhelper.RunGit(t, "tag", wantTag)
	cloneDir := t.TempDir()
	t.Chdir(cloneDir)
	testhelper.RunGit(t, "clone", remoteDir, ".")
	testhelper.RunGit(t, "remote", "rename", "origin", config.RemoteUpstream)
	testhelper.ConfigNewGitRepository(t)
}

func TestBumpWithNestedOutput(t *testing.T) {
	const tag = "bump-nested-output"
	testhelper.RequireCommand(t, "git")
	remoteDir := t.TempDir()
	testhelper.ContinueInNewGitRepository(t, remoteDir)

	pkgRoot := "pkgs/swift-google-auth"
	genDir := path.Join(pkgRoot, "Sources", "GoogleAuth", "generated")
	if err := os.MkdirAll(genDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path.Join(pkgRoot, "Package.swift"), []byte("// Package.swift\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	versionFile := path.Join(genDir, "PackageVersion.swift")
	initialContent := "enum PackageVersion {\n  static let version: Swift.String = \"0.0.0-preview\"\n}\n"
	if err := os.WriteFile(versionFile, []byte(initialContent), 0o644); err != nil {
		t.Fatal(err)
	}
	testhelper.RunGit(t, "add", ".")
	testhelper.RunGit(t, "commit", "-m", "initial version")
	testhelper.RunGit(t, "tag", tag)

	cloneDir := t.TempDir()
	t.Chdir(cloneDir)
	testhelper.RunGit(t, "clone", remoteDir, ".")
	testhelper.RunGit(t, "remote", "rename", "origin", config.RemoteUpstream)
	testhelper.ConfigNewGitRepository(t)

	lib := &config.Library{
		Name:    "google-auth",
		Version: "0.0.0-preview",
		Output:  genDir,
	}

	if err := Bump(t.Context(), lib, genDir, "0.1.0-preview", "git", tag); err != nil {
		t.Fatalf("Bump() error = %v", err)
	}
	if lib.Version != "0.1.0-preview" {
		t.Errorf("got lib.Version = %q, want %q", lib.Version, "0.1.0-preview")
	}

	updatedContent := "enum PackageVersion {\n  static let version: Swift.String = \"0.1.0-preview\"\n}\n"
	if err := os.WriteFile(path.Join(genDir, "PackageVersion.swift"), []byte(updatedContent), 0o644); err != nil {
		t.Fatal(err)
	}
	testhelper.RunGit(t, "commit", "-am", "bump version to 0.1.0-preview")

	lib.Version = "0.1.0-preview"
	if err := Bump(t.Context(), lib, genDir, "0.2.0-preview", "git", tag); err != nil {
		t.Fatalf("Bump() error = %v", err)
	}
	if lib.Version != "0.1.0-preview" {
		t.Errorf("expected version to remain unchanged, got %q", lib.Version)
	}
}

func TestBumpMissingSourcesDir(t *testing.T) {
	lib := &config.Library{
		Name:    "nonexistent",
		Version: "0.1.0-preview",
		Output:  t.TempDir(),
	}
	if err := Bump(t.Context(), lib, lib.Output, "0.2.0-preview", "git", "v1.0.0"); err == nil {
		t.Errorf("Bump() expected error for missing version manifest, got nil")
	}
}

func TestBumpTypeOnlyPackage(t *testing.T) {
	const tag = "bump-type-only-package"
	testhelper.RequireCommand(t, "git")
	remoteDir := t.TempDir()
	testhelper.ContinueInNewGitRepository(t, remoteDir)

	pkgRoot := "generated/swift-google-type"
	sourcesDir := path.Join(pkgRoot, "Sources", "GoogleType")
	if err := os.MkdirAll(sourcesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	versionFile := path.Join(sourcesDir, packageVersionManifest)
	initialContent := "enum PackageVersion {\n  static let version: Swift.String = \"0.2.0\"\n}\n"
	if err := os.WriteFile(versionFile, []byte(initialContent), 0o644); err != nil {
		t.Fatal(err)
	}
	testhelper.RunGit(t, "add", ".")
	testhelper.RunGit(t, "commit", "-m", "initial type-only package")
	testhelper.RunGit(t, "tag", tag)

	cloneDir := t.TempDir()
	t.Chdir(cloneDir)
	testhelper.RunGit(t, "clone", remoteDir, ".")
	testhelper.RunGit(t, "remote", "rename", "origin", config.RemoteUpstream)
	testhelper.ConfigNewGitRepository(t)

	lib := &config.Library{
		Name:    "google-type",
		Version: "0.2.0",
		Output:  pkgRoot,
	}

	if err := Bump(t.Context(), lib, pkgRoot, "0.3.0", "git", tag); err != nil {
		t.Fatalf("Bump() error = %v", err)
	}
	if lib.Version != "0.3.0" {
		t.Errorf("got lib.Version = %q, want %q", lib.Version, "0.3.0")
	}

	// Simulate regeneration with the new version and committing it
	updatedContent := "enum PackageVersion {\n  static let version: Swift.String = \"0.3.0\"\n}\n"
	if err := os.WriteFile(path.Join(pkgRoot, "Sources", "GoogleType", packageVersionManifest), []byte(updatedContent), 0o644); err != nil {
		t.Fatal(err)
	}
	testhelper.RunGit(t, "commit", "-am", "bump version to 0.3.0")

	// Running Bump again should be idempotent (no change to version)
	lib.Version = "0.3.0"
	if err := Bump(t.Context(), lib, pkgRoot, "0.4.0", "git", tag); err != nil {
		t.Fatalf("Bump() error = %v", err)
	}
	if lib.Version != "0.3.0" {
		t.Errorf("expected version to remain 0.3.0 (idempotent), got %q", lib.Version)
	}
}

func TestVersionAlreadyBumpedPackageVersion(t *testing.T) {
	const tag = "package-version-test"
	testhelper.RequireCommand(t, "git")
	remoteDir := t.TempDir()
	testhelper.ContinueInNewGitRepository(t, remoteDir)

	genDir := "Sources/GoogleAuth/generated"
	if err := os.MkdirAll(genDir, 0o755); err != nil {
		t.Fatal(err)
	}
	versionFile := path.Join(genDir, "PackageVersion.swift")
	if err := os.WriteFile(versionFile, []byte("enum PackageVersion {\n  static let version: Swift.String = \"0.0.0-preview\"\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testhelper.RunGit(t, "add", ".")
	testhelper.RunGit(t, "commit", "-m", "initial")
	testhelper.RunGit(t, "tag", tag)

	cloneDir := t.TempDir()
	t.Chdir(cloneDir)
	testhelper.RunGit(t, "clone", remoteDir, ".")
	testhelper.RunGit(t, "remote", "rename", "origin", config.RemoteUpstream)
	testhelper.ConfigNewGitRepository(t)

	bumped, err := versionAlreadyBumped(t.Context(), "git", tag, versionFile)
	if err != nil {
		t.Fatal(err)
	}
	if bumped {
		t.Errorf("expected versionAlreadyBumped == false, got true")
	}

	if err := os.WriteFile(versionFile, []byte("enum PackageVersion {\n  static let version: Swift.String = \"0.1.0-preview\"\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testhelper.RunGit(t, "commit", "-am", "bump")

	bumped, err = versionAlreadyBumped(t.Context(), "git", tag, versionFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bumped {
		t.Errorf("expected versionAlreadyBumped == true, got false")
	}
}
