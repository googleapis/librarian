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

package java

import (
	"encoding/xml"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/filesystem"
)

func TestPostGenerate(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// Copy testdata to tmpDir
	testdataDir := filepath.Join(originalWd, "testdata", "postgenerate")
	if err := copyDir(testdataDir, tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Chdir(tmpDir)
	cfg := &config.Config{
		Language: "java",
		Libraries: []*config.Library{
			{Name: rootLibrary, Version: "1.2.3"},
			{Name: "analytics-admin", Version: "0.98.0"},
			{Name: "area120-tables", Version: "0.92.0"},
			{Name: "aiplatform", Version: "3.89.0"},
		},
	}
	if err := PostGenerate(t.Context(), ".", cfg); err != nil {
		t.Fatal(err)
	}
	// Verify root pom.xml
	rootPom, err := os.ReadFile("pom.xml")
	if err != nil {
		t.Fatal(err)
	}
	rootPomContent := string(rootPom)
	if !strings.Contains(rootPomContent, "<version>0.201.0</version>") {
		t.Errorf("root pom.xml missing correct version, got:\n%s", rootPomContent)
	}
	modules := []string{"java-analytics-admin", "java-area120-tables", "java-aiplatform", "java-grafeas", "java-dns", "java-notification"}
	for _, mod := range modules {
		if !strings.Contains(rootPomContent, "<module>"+mod+"</module>") {
			t.Errorf("root pom.xml missing module %s", mod)
		}
	}
	// Unmarshal and verify gapic-libraries-bom/pom.xml
	wantDeps := []bomDependency{
		{GroupID: "com.google.analytics", ArtifactID: "google-analytics-admin-bom", Version: "0.98.0", Type: "pom", Scope: "import"},
		{GroupID: "com.google.area120", ArtifactID: "google-area120-tables-bom", Version: "0.92.0", Type: "pom", Scope: "import"},
		{GroupID: "com.google.cloud", ArtifactID: "google-cloud-aiplatform-bom", Version: "3.89.0", Type: "pom", Scope: "import"},
		{GroupID: "com.google.cloud", ArtifactID: "google-cloud-dns", Version: "2.86.0", Type: "", Scope: ""},
		{GroupID: "com.google.cloud", ArtifactID: "google-cloud-notification", Version: "0.206.0", Type: "", Scope: ""},
		{GroupID: "io.grafeas", ArtifactID: "grafeas", Version: "1.2.3", Type: "", Scope: ""},
	}
	verifyBOM(t, filepath.Join(gapicBOM, "pom.xml"), "1.2.3", wantDeps)
	// Verify annotations are present in the raw XML
	bomPom, err := os.ReadFile(filepath.Join(gapicBOM, "pom.xml"))
	if err != nil {
		t.Fatal(err)
	}
	bomPomContent := string(bomPom)
	if !strings.Contains(bomPomContent, "<!-- {x-version-update:google-cloud-aiplatform:current} -->") {
		t.Errorf("%s/pom.xml missing annotation google-cloud-aiplatform", gapicBOM)
	}
}

type bomDependency struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
	Type       string `xml:"type"`
	Scope      string `xml:"scope"`
}

type bomPom struct {
	Version      string          `xml:"version"`
	Dependencies []bomDependency `xml:"dependencyManagement>dependencies>dependency"`
}

func verifyBOM(t *testing.T, path string, wantVersion string, wantDeps []bomDependency) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var p bomPom
	if err := xml.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	if p.Version != wantVersion {
		t.Errorf("%s version = %q, want %q", path, p.Version, wantVersion)
	}
	// Filter out dependencies we are not testing (there might be others from searchForBOMArtifacts)
	var gotDeps []bomDependency
	for _, d := range p.Dependencies {
		for _, w := range wantDeps {
			if d.ArtifactID == w.ArtifactID {
				gotDeps = append(gotDeps, d)
				break
			}
		}
	}

	if diff := cmp.Diff(wantDeps, gotDeps); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	// Verify that libraries like java-maps-places are excluded because their
	// GroupID (com.google.maps) is not in the allowed groupInclusions list.
	if slices.ContainsFunc(p.Dependencies, func(d bomDependency) bool {
		return d.ArtifactID == "google-maps-places-bom"
	}) {
		t.Errorf("%s should NOT contain google-maps-places-bom", path)
	}
}

func TestSearchForJavaModules(t *testing.T) {
	tmpDir := t.TempDir()
	// Setup: mix of modules, non-modules, and excluded directories
	dirs := []string{
		"module-b",
		"module-a",
		"not-a-module",
		gapicBOM,
		"google-cloud-shared-deps",
	}
	for _, dir := range dirs {
		if err := os.Mkdir(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}
	// Add pom.xml to modules (including an excluded one to verify filtering)
	for _, mod := range []string{"module-a", "module-b", gapicBOM} {
		if err := os.WriteFile(filepath.Join(tmpDir, mod, "pom.xml"), []byte("<project/>"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := searchForJavaModules(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"module-a", "module-b"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestSearchForJavaModules_Error(t *testing.T) {
	tmpDir := t.TempDir()
	// Make directory unreadable to cause os.ReadDir failure
	if err := os.Chmod(tmpDir, 0000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(tmpDir, 0755)
	_, err := searchForJavaModules(tmpDir)
	if err == nil {
		t.Error("searchForJavaModules expected error, got nil")
	}
}

func TestPostGenerate_SearchError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	// Make directory unreadable to cause searchForJavaModules failure
	if err := os.Chmod(tmpDir, 0000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(tmpDir, 0755)
	cfg := &config.Config{
		Libraries: []*config.Library{
			{Name: rootLibrary, Version: "1.2.3"},
		},
	}
	err := PostGenerate(t.Context(), ".", cfg)
	if !errors.Is(err, errModuleDiscovery) {
		t.Errorf("got error %v, want %v", err, errModuleDiscovery)
	}
}

func TestPostGenerate_Error(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	// Make directory read-only to cause os.Create("pom.xml") failure
	if err := os.Chmod(tmpDir, 0555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(tmpDir, 0755)
	cfg := &config.Config{
		Libraries: []*config.Library{
			{Name: rootLibrary, Version: "1.2.3"},
		},
	}
	err := PostGenerate(t.Context(), ".", cfg)
	if !errors.Is(err, errRootPomGeneration) {
		t.Errorf("got error %v, want %v", err, errRootPomGeneration)
	}
}

func TestExtractBOMConfig_Error(t *testing.T) {
	tmpDir := t.TempDir()
	for _, test := range []struct {
		name    string
		library string
		bom     string
		pom     string
		wantErr error
	}{
		{
			name:    "missing pom.xml",
			library: "lib",
			bom:     "lib-bom",
			wantErr: fs.ErrNotExist,
		},
		{
			name:    "invalid xml",
			library: "lib",
			bom:     "lib-bom",
			pom:     "<project xmlns=\"http://maven.apache.org/POM/4.0.0\">invalid",
			wantErr: errMalformedBOM,
		},
		{
			name:    "invalid artifact id (no dash)",
			library: "lib",
			bom:     "lib-bom",
			pom:     "<project xmlns=\"http://maven.apache.org/POM/4.0.0\"><artifactId>nodash</artifactId></project>",
			wantErr: errInvalidBOMArtifactID,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.pom != "" {
				dir := filepath.Join(tmpDir, test.library, test.bom)
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(test.pom), 0644); err != nil {
					t.Fatal(err)
				}
			}
			_, err := extractBOMConfig(tmpDir, test.library, test.bom)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("got error %v, want %v", err, test.wantErr)
			}
		})
	}
}

func copyDir(src, dest string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return filesystem.CopyFile(path, target)
	})
}
