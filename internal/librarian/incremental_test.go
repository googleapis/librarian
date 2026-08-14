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

package librarian

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sources"
)

func TestFingerprintFilesChangesWithContent(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.proto")
	if err := os.WriteFile(a, []byte("message A {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	fp1, err := fingerprintFiles([]string{a}, "extra")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(a, []byte("message A { int32 x = 1; }"), 0o644); err != nil {
		t.Fatal(err)
	}
	fp2, err := fingerprintFiles([]string{a}, "extra")
	if err != nil {
		t.Fatal(err)
	}
	if fp1 == fp2 {
		t.Error("fingerprint unchanged after editing a proto")
	}
}

func TestFingerprintFilesOrderIndependentAndExtra(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.proto")
	b := filepath.Join(dir, "b.proto")
	if err := os.WriteFile(a, []byte("A"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("B"), 0o644); err != nil {
		t.Fatal(err)
	}
	fpAB, err := fingerprintFiles([]string{a, b}, "x")
	if err != nil {
		t.Fatal(err)
	}
	fpBA, err := fingerprintFiles([]string{b, a}, "x")
	if err != nil {
		t.Fatal(err)
	}
	if fpAB != fpBA {
		t.Error("fingerprint should be independent of file order")
	}
	fpExtra, err := fingerprintFiles([]string{a, b}, "y")
	if err != nil {
		t.Fatal(err)
	}
	if fpAB == fpExtra {
		t.Error("changing extra should change the fingerprint")
	}
}

func TestGeneratedManifestRoundTrip(t *testing.T) {
	out := t.TempDir()
	const fp = "deadbeef"
	if generatedUpToDate(out, fp) {
		t.Error("upToDate before any manifest should be false")
	}
	if err := writeGeneratedManifest(out, fp); err != nil {
		t.Fatal(err)
	}
	if !generatedUpToDate(out, fp) {
		t.Error("upToDate should be true after writing a matching manifest")
	}
	if generatedUpToDate(out, "other") {
		t.Error("upToDate should be false for a different fingerprint")
	}
	if generatedUpToDate(out, "") {
		t.Error("empty fingerprint should never be up to date")
	}
	// Writing an empty fingerprint is a no-op (no manifest created).
	empty := t.TempDir()
	if err := writeGeneratedManifest(empty, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(generatedManifestPath(empty)); !os.IsNotExist(err) {
		t.Error("empty fingerprint should not create a manifest file")
	}
}

// newTestSources creates a googleapis source root under a temp dir and writes a
// proto for each api path, returning the sources and the root dir.
func newTestSources(t *testing.T, apiPaths ...string) (*sources.Sources, string) {
	t.Helper()
	root := t.TempDir()
	for _, p := range apiPaths {
		full := filepath.Join(root, filepath.FromSlash(p), "service.proto")
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("service "+p), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return &sources.Sources{Googleapis: root}, root
}

func TestLibraryInputFingerprintChanges(t *testing.T) {
	src, root := newTestSources(t, "google/cloud/foo/v1")
	lib := &config.Library{
		Name:   "java-foo",
		Output: t.TempDir(),
		APIs:   []*config.API{{Path: "google/cloud/foo/v1"}},
	}
	fp1, err := libraryInputFingerprint(lib, src, "extra")
	if err != nil {
		t.Fatalf("fingerprint: %v", err)
	}
	if fp1 == "" {
		t.Fatal("empty fingerprint")
	}
	// Editing an input proto must change the fingerprint.
	proto := filepath.Join(root, "google/cloud/foo/v1/service.proto")
	if err := os.WriteFile(proto, []byte("service changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	fp2, err := libraryInputFingerprint(lib, src, "extra")
	if err != nil {
		t.Fatal(err)
	}
	if fp1 == fp2 {
		t.Error("fingerprint unchanged after editing an input proto")
	}
}

func TestFilterChangedLibraries(t *testing.T) {
	src, _ := newTestSources(t, "google/cloud/foo/v1", "google/cloud/bar/v1")
	foo := &config.Library{Name: "java-foo", Output: t.TempDir(), APIs: []*config.API{{Path: "google/cloud/foo/v1"}}}
	bar := &config.Library{Name: "java-bar", Output: t.TempDir(), APIs: []*config.API{{Path: "google/cloud/bar/v1"}}}

	// Record foo's current fingerprint so it looks already-generated.
	fooFP, err := libraryInputFingerprint(foo, src, "extra")
	if err != nil {
		t.Fatal(err)
	}
	if err := writeGeneratedManifest(foo.Output, fooFP); err != nil {
		t.Fatal(err)
	}

	kept, fps := filterChangedLibraries([]*config.Library{foo, bar}, src, "extra")
	if len(kept) != 1 || kept[0].Name != "java-bar" {
		t.Fatalf("kept = %v, want only java-bar (foo is unchanged)", names(kept))
	}
	if fps["java-bar"] == "" {
		t.Error("expected a fingerprint recorded for the kept library")
	}
	if _, ok := fps["java-foo"]; ok {
		t.Error("skipped library should not have a fingerprint entry")
	}
}

func names(libs []*config.Library) []string {
	out := make([]string, len(libs))
	for i, l := range libs {
		out[i] = l.Name
	}
	return out
}
