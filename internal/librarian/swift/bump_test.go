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
	return path.Join(testPackageName, "Sources", testLibraryName, manifestFile)
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

	name := path.Join(testPackageName, "Sources", "GoogleCloudNew", manifestFile)
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
	name := path.Join("not-the-right-package", "Sources", "NotTheRightLibrary", manifestFile)
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
