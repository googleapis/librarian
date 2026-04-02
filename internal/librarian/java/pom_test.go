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
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

// update is used to refresh the golden files in testdata/ when template
// changes result in intentional output differences.
// Usage: go test ./internal/librarian/java -v -update.
var update = flag.Bool("update", false, "update golden files")

func TestSyncPoms_Golden(t *testing.T) {
	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	testdataDir := filepath.Join("testdata", "syncpoms", "secretmanager-v1")
	library := &config.Library{
		Name:    "secretmanager",
		Version: "1.2.3",
		APIs: []*config.API{
			{Path: "google/cloud/secretmanager/v1"},
		},
	}
	tmpDir := t.TempDir()
	// Pre-create the directories that generatePomsIfMissing expects to exist.
	protoArtifactID := "proto-google-cloud-secretmanager-v1"
	grpcArtifactID := "grpc-google-cloud-secretmanager-v1"
	gapicArtifactID := "google-cloud-secretmanager"
	bomArtifactID := "google-cloud-secretmanager-bom"
	for _, artifact := range []string{protoArtifactID, grpcArtifactID, gapicArtifactID, bomArtifactID} {
		if err := os.MkdirAll(filepath.Join(tmpDir, artifact), 0755); err != nil {
			t.Fatal(err)
		}
	}
	metadata := &repoMetadata{
		NamePretty:     "Secret Manager",
		APIDescription: "Stores sensitive data such as API keys, passwords, and certificates.\nProvides convenience while improving security.",
	}
	if err := syncPoms(library, tmpDir, googleapisDir, "1.2.3", metadata); err != nil {
		t.Fatal(err)
	}
	artifacts := []string{protoArtifactID, grpcArtifactID, gapicArtifactID, "google-cloud-secretmanager-bom", "google-cloud-secretmanager-parent"}
	for _, artifact := range artifacts {
		dir := artifact
		if artifact == "google-cloud-secretmanager-parent" {
			dir = ""
		}
		gotPath := filepath.Join(tmpDir, dir, "pom.xml")
		got, err := os.ReadFile(gotPath)
		if err != nil {
			t.Fatal(err)
		}
		goldenPath := filepath.Join(testdataDir, artifact, "pom.xml")
		if *update {
			if err := os.MkdirAll(filepath.Dir(goldenPath), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(goldenPath, got, 0644); err != nil {
				t.Fatal(err)
			}
		}
		want, err := os.ReadFile(goldenPath)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(string(want), string(got)); diff != "" {
			t.Errorf("mismatch in %s (-want +got):\n%s\n\nHint: run 'go test ./internal/librarian/java -v -update' to update golden files.", artifact, diff)
		}
	}
}

func TestSyncPoms_Update(t *testing.T) {
	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	tmpDir := t.TempDir()
	gapicArtifactID := "google-cloud-secretmanager"
	protoArtifactID := "proto-google-cloud-secretmanager-v1"
	grpcArtifactID := "grpc-google-cloud-secretmanager-v1"
	bomArtifactID := "google-cloud-secretmanager-bom"
	for _, artifact := range []string{protoArtifactID, grpcArtifactID, gapicArtifactID, bomArtifactID} {
		if err := os.MkdirAll(filepath.Join(tmpDir, artifact), 0755); err != nil {
			t.Fatal(err)
		}
	}

	gapicDir := filepath.Join(tmpDir, gapicArtifactID)

	pomPath := filepath.Join(gapicDir, "pom.xml")
	initialContent := `<?xml version='1.0' encoding='UTF-8'?>
<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd">
  <modelVersion>4.0.0</modelVersion>
  <groupId>com.google.cloud</groupId>
  <artifactId>google-cloud-secretmanager</artifactId>
  <version>1.2.3</version><!-- {x-version-update:google-cloud-secretmanager:current} -->
  <dependencies>
    <!-- {x-generated-proto-dependencies-start} -->
    <dependency>
      <groupId>com.google.api.grpc</groupId>
      <artifactId>proto-google-cloud-secretmanager-v0</artifactId>
    </dependency>
    <!-- {x-generated-proto-dependencies-end} -->
    <dependency>
      <groupId>com.google.guava</groupId>
      <artifactId>guava</artifactId>
    </dependency>
    <!-- {x-generated-grpc-dependencies-start} -->
    <dependency>
      <groupId>com.google.api.grpc</groupId>
      <artifactId>grpc-google-cloud-secretmanager-v0</artifactId>
      <scope>test</scope>
    </dependency>
    <!-- {x-generated-grpc-dependencies-end} -->
  </dependencies>
</project>`
	if err := os.WriteFile(pomPath, []byte(initialContent), 0644); err != nil {
		t.Fatal(err)
	}

	library := &config.Library{
		Name:    "secretmanager",
		Version: "1.2.3",
		APIs: []*config.API{
			{Path: "google/cloud/secretmanager/v1"},
		},
	}
	metadata := &repoMetadata{
		NamePretty:     "Secret Manager",
		APIDescription: "Description",
	}

	if err := syncPoms(library, tmpDir, googleapisDir, "1.2.3", metadata); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(pomPath)
	if err != nil {
		t.Fatal(err)
	}
	gotStr := string(got)

	if !strings.Contains(gotStr, "proto-google-cloud-secretmanager-v1") {
		t.Errorf("missing updated proto dependency, got:\n%s", gotStr)
	}
	if strings.Contains(gotStr, "proto-google-cloud-secretmanager-v0") {
		t.Errorf("still contains old proto dependency, got:\n%s", gotStr)
	}
	if !strings.Contains(gotStr, "grpc-google-cloud-secretmanager-v1") {
		t.Errorf("missing updated grpc dependency, got:\n%s", gotStr)
	}
	if !strings.Contains(gotStr, "<!-- {x-version-update:google-cloud-secretmanager:current} -->") {
		t.Error("lost version update comment")
	}
	if !strings.Contains(gotStr, "<artifactId>guava</artifactId>") {
		t.Error("lost guava dependency")
	}
}

func TestCollectModules_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
	}{
		{
			name: "invalid distribution name",
			library: &config.Library{
				Java: &config.JavaModule{
					DistributionNameOverride: "invalid-name",
				},
			},
		},
		{
			name: "failed to find api config",
			library: &config.Library{
				APIs: []*config.API{
					{Path: "google/ads/unrecognized/v1"},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := collectModules(test.library, t.TempDir(), "/nonexistent", "1.2.3", &repoMetadata{}); err == nil {
				t.Error("collectModules() error = nil, want non-nil")
			}
		})
	}
}

func TestIsPomMissing(t *testing.T) {
	for _, test := range []struct {
		name  string
		setup func(t *testing.T) string
		want  bool
	}{
		{
			name: "pom exists",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				if err := os.WriteFile(filepath.Join(dir, "pom.xml"), []byte("content"), 0644); err != nil {
					t.Fatal(err)
				}
				return dir
			},
		},
		{
			name: "pom missing",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			want: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := test.setup(t)
			got, err := isPomMissing(dir)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Errorf("isPomMissing(%q) = %v, want %v", dir, got, test.want)
			}
		})
	}
}

func TestIsPomMissing_DirMissingError(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent")
	_, err := isPomMissing(dir)
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("isPomMissing(%q) error = %v, want %v", dir, err, os.ErrNotExist)
	}
}

func TestProtoGroupID(t *testing.T) {
	for _, test := range []struct {
		name                string
		mainArtifactGroupID string
		want                string
	}{
		{
			name:                "cloud group id",
			mainArtifactGroupID: "com.google.cloud",
			want:                "com.google.api.grpc",
		},
		{
			name:                "analytics group id",
			mainArtifactGroupID: "com.google.analytics",
			want:                "com.google.api.grpc",
		},
		{
			name:                "area120 group id",
			mainArtifactGroupID: "com.google.area120",
			want:                "com.google.api.grpc",
		},
		{
			name:                "non-cloud group id",
			mainArtifactGroupID: "com.google.maps",
			want:                "com.google.maps.api.grpc",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := protoGroupID(test.mainArtifactGroupID)
			if got != test.want {
				t.Errorf("protoGroupID(%q) = %q, want %q", test.mainArtifactGroupID, got, test.want)
			}
		})
	}
}
