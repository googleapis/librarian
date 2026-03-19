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
	"errors"
	"os"
	"path/filepath"
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
			{Name: "google-cloud-java", Version: "1.2.3"},
			{Name: "analytics-admin", Version: "0.98.0"},
			{Name: "area120-tables", Version: "0.92.0"},
			{Name: "aiplatform", Version: "3.89.0"},
		},
	}
	if err := PostGenerate(t.Context(), cfg); err != nil {
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
	// Verify gapic-libraries-bom/pom.xml
	bomPom, err := os.ReadFile(filepath.Join("gapic-libraries-bom", "pom.xml"))
	if err != nil {
		t.Fatal(err)
	}
	bomPomContent := string(bomPom)
	if !strings.Contains(bomPomContent, "<version>1.2.3</version>") {
		t.Errorf("gapic-libraries-bom/pom.xml missing correct version, got:\n%s", bomPomContent)
	}
	// Verify inclusions
	if !strings.Contains(bomPomContent, "<artifactId>google-analytics-admin-bom</artifactId>") {
		t.Errorf("gapic-libraries-bom/pom.xml missing artifact google-analytics-admin-bom")
	}
	if !strings.Contains(bomPomContent, "<version>0.98.0</version>") {
		t.Errorf("gapic-libraries-bom/pom.xml missing analytics-admin version 0.98.0")
	}
	if !strings.Contains(bomPomContent, "<artifactId>google-area120-tables-bom</artifactId>") {
		t.Errorf("gapic-libraries-bom/pom.xml missing artifact google-area120-tables-bom")
	}
	if !strings.Contains(bomPomContent, "<version>0.92.0</version>") {
		t.Errorf("gapic-libraries-bom/pom.xml missing area120-tables version 0.92.0")
	}
	if !strings.Contains(bomPomContent, "<artifactId>google-cloud-aiplatform-bom</artifactId>") {
		t.Errorf("gapic-libraries-bom/pom.xml missing artifact google-cloud-aiplatform-bom")
	}
	if !strings.Contains(bomPomContent, "<version>3.89.0</version>") {
		t.Errorf("gapic-libraries-bom/pom.xml missing aiplatform version 3.89.0")
	}
	// java-maps-places should be excluded because of GroupID com.google.maps
	if strings.Contains(bomPomContent, "google-maps-places-bom") {
		t.Errorf("gapic-libraries-bom/pom.xml should NOT contain google-maps-places-bom")
	}
	// Verify special case java-grafeas
	if !strings.Contains(bomPomContent, "<artifactId>grafeas</artifactId>") {
		t.Errorf("gapic-libraries-bom/pom.xml missing artifact grafeas")
	}
	if !strings.Contains(bomPomContent, "<version>1.2.3</version>") {
		t.Errorf("gapic-libraries-bom/pom.xml missing grafeas version 1.2.3")
	}
	// It should NOT have type pom and scope import
	grafeasEntry := `<dependency>
        <groupId>io.grafeas</groupId>
        <artifactId>grafeas</artifactId>
        <version>1.2.3</version><!-- {x-version-update:grafeas:current} -->
      </dependency>`
	if !strings.Contains(bomPomContent, grafeasEntry) {
		t.Errorf("gapic-libraries-bom/pom.xml missing expected grafeas entry, got:\n%s", bomPomContent)
	}
	// Verify annotations
	if !strings.Contains(bomPomContent, "<!-- {x-version-update:google-cloud-aiplatform:current} -->") {
		t.Errorf("gapic-libraries-bom/pom.xml missing annotation google-cloud-aiplatform, got:\n%s", bomPomContent)
	}
}

func TestSearchForJavaModules(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	// Setup: mix of modules, non-modules, and excluded directories
	dirs := []string{
		"module-b",
		"module-a",
		"not-a-module",
		"gapic-libraries-bom",
		"google-cloud-shared-deps",
	}
	for _, dir := range dirs {
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	// Add pom.xml to modules (including an excluded one to verify filtering)
	for _, mod := range []string{"module-a", "module-b", "gapic-libraries-bom"} {
		if err := os.WriteFile(filepath.Join(mod, "pom.xml"), []byte("<project/>"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := searchForJavaModules()
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
	t.Chdir(tmpDir)
	// Make directory unreadable to cause os.ReadDir failure
	if err := os.Chmod(tmpDir, 0000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(tmpDir, 0755)
	_, err := searchForJavaModules()
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
			{Name: "google-cloud-java", Version: "1.2.3"},
		},
	}
	err := PostGenerate(t.Context(), cfg)
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
			{Name: "google-cloud-java", Version: "1.2.3"},
		},
	}
	err := PostGenerate(t.Context(), cfg)
	if !errors.Is(err, errRootPomGeneration) {
		t.Errorf("got error %v, want %v", err, errRootPomGeneration)
	}
}

func TestExtractBOMConfig_Error(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	for _, test := range []struct {
		name    string
		library string
		bom     string
		pom     string
		wantErr string
	}{
		{
			name:    "missing pom.xml",
			library: "lib",
			bom:     "lib-bom",
			wantErr: "no such file or directory",
		},
		{
			name:    "invalid xml",
			library: "lib",
			bom:     "lib-bom",
			pom:     "<project xmlns=\"http://maven.apache.org/POM/4.0.0\">invalid",
			wantErr: "XML syntax error",
		},
		{
			name:    "invalid artifact id (no dash)",
			library: "lib",
			bom:     "lib-bom",
			pom:     "<project xmlns=\"http://maven.apache.org/POM/4.0.0\"><artifactId>nodash</artifactId></project>",
			wantErr: "expected at least one dash",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.pom != "" {
				dir := filepath.Join(test.library, test.bom)
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(test.pom), 0644); err != nil {
					t.Fatal(err)
				}
			}
			_, err := extractBOMConfig(test.library, test.bom)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("got error %v, want to contain %q", err, test.wantErr)
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
