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

package maven

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

// These unit tests exercise the install skip-check logic directly, without
// executing mvn, so they run on every platform (the integration-level skip
// tests need a shell stub and therefore only run on POSIX/CI).

func TestLocalFingerprintChangesWithContent(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "pom.xml")
	if err := os.WriteFile(src, []byte("<project>one</project>"), 0o644); err != nil {
		t.Fatal(err)
	}
	fp1, err := localFingerprint(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("<project>a-different-length</project>"), 0o644); err != nil {
		t.Fatal(err)
	}
	fp2, err := localFingerprint(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fp1 == fp2 {
		t.Error("fingerprint did not change after editing source")
	}
}

func TestLocalFingerprintIgnoresTargetDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pom.xml"), []byte("<project/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	before, err := localFingerprint(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate a build writing into target/: this must not change the fingerprint.
	targetDir := filepath.Join(dir, "target")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "artifact.jar"), []byte("built"), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := localFingerprint(dir)
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Errorf("build output under target/ changed the fingerprint: %s -> %s", before, after)
	}
}

func TestExternalFingerprintDependsOnCoordinate(t *testing.T) {
	base := &config.MavenTool{Name: "gjf", GroupID: "g", ArtifactID: "a", Version: "1.0.0", Packaging: "jar"}
	same := &config.MavenTool{Name: "gjf", GroupID: "g", ArtifactID: "a", Version: "1.0.0", Packaging: "jar"}
	bumped := &config.MavenTool{Name: "gjf", GroupID: "g", ArtifactID: "a", Version: "2.0.0", Packaging: "jar"}
	if externalFingerprint(base) != externalFingerprint(same) {
		t.Error("same coordinate should yield the same fingerprint")
	}
	if externalFingerprint(base) == externalFingerprint(bumped) {
		t.Error("different version should yield a different fingerprint")
	}
}

func TestUpToDate(t *testing.T) {
	binDir := t.TempDir()
	const name, fp = "protoc-gen-java_gapic", "local:abc123"

	// No wrapper, no marker: not up to date.
	if upToDate(binDir, name, fp) {
		t.Error("upToDate should be false before install")
	}
	// Wrapper present but no marker: not up to date.
	if err := os.WriteFile(filepath.Join(binDir, name), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if upToDate(binDir, name, fp) {
		t.Error("upToDate should be false with wrapper but no marker")
	}
	// Marker written with matching fingerprint: up to date.
	if err := writeMarker(binDir, name, fp); err != nil {
		t.Fatal(err)
	}
	if !upToDate(binDir, name, fp) {
		t.Error("upToDate should be true with wrapper + matching marker")
	}
	// Different fingerprint (source changed): not up to date.
	if upToDate(binDir, name, "local:different") {
		t.Error("upToDate should be false when the fingerprint differs")
	}
	// Empty fingerprint never matches.
	if upToDate(binDir, name, "") {
		t.Error("empty fingerprint should never be up to date")
	}
	// Wrapper removed but marker remains: not up to date.
	if err := os.Remove(filepath.Join(binDir, name)); err != nil {
		t.Fatal(err)
	}
	if upToDate(binDir, name, fp) {
		t.Error("upToDate should be false when the wrapper is missing")
	}
}
