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
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

// update is used to refresh the golden files in testdata/ when template
// changes result in intentional output differences.
// Usage: go test ./internal/librarian/java -v -update.
var update = flag.Bool("update", false, "update golden files")

func TestSyncPOMs_Golden(t *testing.T) {
	testdataDir := filepath.Join("testdata", "syncpoms", "secretmanager-v1")
	library := &config.Library{
		Name:    "secretmanager",
		Version: "1.2.3",
		APIs: []*config.API{
			{Path: "google/cloud/secretmanager/v1"},
		},
	}
	apiPath := library.APIs[0].Path
	transports := map[string]serviceconfig.Transport{
		apiPath: serviceconfig.GRPC,
	}
	tmpDir := t.TempDir()
	// Pre-create the directories that generatePOMsIfMissing expects to exist.
	protoArtifactID := "proto-google-cloud-secretmanager-v1"
	gRPCArtifactID := "grpc-google-cloud-secretmanager-v1"
	gapicArtifactID := "google-cloud-secretmanager"
	bomArtifactID := "google-cloud-secretmanager-bom"
	for _, artifact := range []string{protoArtifactID, gRPCArtifactID, gapicArtifactID, bomArtifactID} {
		if err := os.MkdirAll(filepath.Join(tmpDir, artifact), 0755); err != nil {
			t.Fatal(err)
		}
	}
	metadata := &repoMetadata{
		NamePretty:     "Secret Manager",
		APIDescription: "Stores sensitive data such as API keys, passwords, and certificates.\nProvides convenience while improving security.",
	}
	if err := syncPOMs(library, tmpDir, "1.2.3", metadata, transports); err != nil {
		t.Fatal(err)
	}
	artifacts := []string{protoArtifactID, gRPCArtifactID, gapicArtifactID, "google-cloud-secretmanager-bom", "google-cloud-secretmanager-parent"}
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

func TestSyncPOMs_Update(t *testing.T) {
	testdataDir := filepath.Join("testdata", "syncpoms", "secretmanager-v1")
	tmpDir := t.TempDir()

	// Setup directory structure for all modules.
	protoArtifactID := "proto-google-cloud-secretmanager-v1"
	gRPCArtifactID := "grpc-google-cloud-secretmanager-v1"
	gapicArtifactID := "google-cloud-secretmanager"
	bomArtifactID := "google-cloud-secretmanager-bom"
	for _, artifact := range []string{protoArtifactID, gRPCArtifactID, gapicArtifactID, bomArtifactID} {
		if err := os.MkdirAll(filepath.Join(tmpDir, artifact), 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Prepare mangled existing POMs for Client, BOM, and Parent to simulate outdated state.
	targets := []struct {
		relPath string
		markers []struct{ start, end string }
	}{
		{
			relPath: filepath.Join(gapicArtifactID, "pom.xml"),
			markers: []struct{ start, end string }{
				{managedProtoStartMarker, managedProtoEndMarker},
				{managedGRPCStartMarker, managedGRPCEndMarker},
			},
		},
		{
			relPath: filepath.Join(bomArtifactID, "pom.xml"),
			markers: []struct{ start, end string }{
				{managedDependenciesStartMarker, managedDependenciesEndMarker},
			},
		},
		{
			relPath: "pom.xml", // Parent
			markers: []struct{ start, end string }{
				{managedDependenciesStartMarker, managedDependenciesEndMarker},
				{managedModulesStartMarker, managedModulesEndMarker},
			},
		},
	}

	for _, target := range targets {
		goldenDir := filepath.Dir(target.relPath)
		if target.relPath == "pom.xml" {
			goldenDir = "google-cloud-secretmanager-parent"
		}
		goldenPath := filepath.Join(testdataDir, goldenDir, "pom.xml")

		content, err := os.ReadFile(goldenPath)
		if err != nil {
			t.Fatal(err)
		}
		mangled := string(content)
		for _, m := range target.markers {
			var err error
			mangled, err = replaceBlock(mangled, m.start, m.end, "      <mangled>true</mangled>")
			if err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(filepath.Join(tmpDir, target.relPath), []byte(mangled), 0644); err != nil {
			t.Fatal(err)
		}
	}

	library := &config.Library{
		Name:    "secretmanager",
		Version: "1.2.3",
		APIs: []*config.API{
			{Path: "google/cloud/secretmanager/v1"},
		},
	}
	transports := map[string]serviceconfig.Transport{
		"google/cloud/secretmanager/v1": serviceconfig.GRPC,
	}
	metadata := &repoMetadata{
		NamePretty:     "Secret Manager",
		APIDescription: "Stores sensitive data such as API keys, passwords, and certificates.\nProvides convenience while improving security.",
	}

	if err := syncPOMs(library, tmpDir, "1.2.3", metadata, transports); err != nil {
		t.Fatal(err)
	}

	// Verify all POMs match their golden versions.
	for _, target := range targets {
		got, err := os.ReadFile(filepath.Join(tmpDir, target.relPath))
		if err != nil {
			t.Fatal(err)
		}
		goldenDir := filepath.Dir(target.relPath)
		if target.relPath == "pom.xml" {
			goldenDir = "google-cloud-secretmanager-parent"
		}
		goldenPath := filepath.Join(testdataDir, goldenDir, "pom.xml")
		want, err := os.ReadFile(goldenPath)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(string(want), string(got)); diff != "" {
			t.Errorf("mismatch in %s (-want +got):\n%s", target.relPath, diff)
		}
	}
}

func TestSyncPOMs_NoUpdate(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup directory structure and existing POMs for ALL modules.
	protoArtifactID := "proto-google-cloud-secretmanager-v1"
	gRPCArtifactID := "grpc-google-cloud-secretmanager-v1"
	gapicArtifactID := "google-cloud-secretmanager"
	bomArtifactID := "google-cloud-secretmanager-bom"
	for _, artifact := range []string{protoArtifactID, gRPCArtifactID, gapicArtifactID, bomArtifactID, ""} {
		dir := filepath.Join(tmpDir, artifact)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		// Write a dummy pom.xml to ensure isMissing is false for all modules.
		if err := os.WriteFile(filepath.Join(dir, "pom.xml"), []byte("<!-- {x-generated-dependencies-start} -->\n      <mangled>true</mangled>\n<!-- {x-generated-dependencies-end} -->"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	library := &config.Library{
		Name:    "secretmanager",
		Version: "1.2.3",
		APIs: []*config.API{
			{Path: "google/cloud/secretmanager/v1"},
		},
	}
	transports := map[string]serviceconfig.Transport{
		"google/cloud/secretmanager/v1": serviceconfig.GRPC,
	}
	metadata := &repoMetadata{
		NamePretty:     "Secret Manager",
		APIDescription: "Stores sensitive data such as API keys, passwords, and certificates.\nProvides convenience while improving security.",
	}

	if err := syncPOMs(library, tmpDir, "1.2.3", metadata, transports); err != nil {
		t.Fatal(err)
	}

	// Verify that the mangled POMs were NOT updated.
	for _, artifact := range []string{gapicArtifactID, bomArtifactID, ""} {
		got, err := os.ReadFile(filepath.Join(tmpDir, artifact, "pom.xml"))
		if err != nil {
			t.Fatal(err)
		}
		want := "<!-- {x-generated-dependencies-start} -->\n      <mangled>true</mangled>\n<!-- {x-generated-dependencies-end} -->"
		if diff := cmp.Diff(string(want), string(got)); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestCollectModules(t *testing.T) {
	for _, test := range []struct {
		name            string
		library         *config.Library
		monorepoVersion string
		metadata        *repoMetadata
		transports      map[string]serviceconfig.Transport
		setup           func(t *testing.T, libraryDir string)
		want            []javaModule
	}{
		{
			name: "single api, grpc transport",
			library: &config.Library{
				Name:    "secretmanager",
				Version: "1.2.3",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			monorepoVersion: "1.2.3",
			metadata: &repoMetadata{
				NamePretty:     "Secret Manager",
				APIDescription: "Secret Manager API",
			},
			transports: map[string]serviceconfig.Transport{
				"google/cloud/secretmanager/v1": serviceconfig.GRPC,
			},
			setup: func(t *testing.T, libraryDir string) {
				dirs := []string{
					"proto-google-cloud-secretmanager-v1",
					"grpc-google-cloud-secretmanager-v1",
					"google-cloud-secretmanager",
					"google-cloud-secretmanager-bom",
					"", // parent
				}
				for _, d := range dirs {
					if err := os.MkdirAll(filepath.Join(libraryDir, d), 0755); err != nil {
						t.Fatal(err)
					}
				}
			},
			want: []javaModule{
				{artifactID: "proto-google-cloud-secretmanager-v1", isMissing: true, template: protoPOMTemplateName},
				{artifactID: "grpc-google-cloud-secretmanager-v1", isMissing: true, template: gRPCPOMTemplateName},
				{artifactID: "google-cloud-secretmanager", isMissing: true, template: clientPOMTemplateName},
				{artifactID: "google-cloud-secretmanager-bom", isMissing: true, template: bomPOMTemplateName},
				{artifactID: "google-cloud-secretmanager-parent", isMissing: true, template: parentPOMTemplateName},
			},
		},
		{
			name: "single api, rest transport skips grpc module",
			library: &config.Library{
				Name:    "secretmanager",
				Version: "1.2.3",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			monorepoVersion: "1.2.3",
			metadata: &repoMetadata{
				NamePretty: "Secret Manager",
			},
			transports: map[string]serviceconfig.Transport{
				"google/cloud/secretmanager/v1": serviceconfig.Rest,
			},
			setup: func(t *testing.T, libraryDir string) {
				dirs := []string{
					"proto-google-cloud-secretmanager-v1",
					"google-cloud-secretmanager",
					"google-cloud-secretmanager-bom",
					"", // parent
				}
				for _, d := range dirs {
					if err := os.MkdirAll(filepath.Join(libraryDir, d), 0755); err != nil {
						t.Fatal(err)
					}
				}
			},
			want: []javaModule{
				{artifactID: "proto-google-cloud-secretmanager-v1", isMissing: true, template: protoPOMTemplateName},
				{artifactID: "google-cloud-secretmanager", isMissing: true, template: clientPOMTemplateName},
				{artifactID: "google-cloud-secretmanager-bom", isMissing: true, template: bomPOMTemplateName},
				{artifactID: "google-cloud-secretmanager-parent", isMissing: true, template: parentPOMTemplateName},
			},
		},
		{
			name: "single api, proto only skips grpc module",
			library: &config.Library{
				Name:    "secretmanager",
				Version: "1.2.3",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
				Java: &config.JavaModule{
					JavaAPIs: []*config.JavaAPI{
						{Path: "google/cloud/secretmanager/v1", ProtoOnly: true},
					},
				},
			},
			monorepoVersion: "1.2.3",
			metadata: &repoMetadata{
				NamePretty: "Secret Manager",
			},
			transports: map[string]serviceconfig.Transport{
				"google/cloud/secretmanager/v1": serviceconfig.GRPC,
			},
			setup: func(t *testing.T, libraryDir string) {
				dirs := []string{
					"proto-google-cloud-secretmanager-v1",
					"google-cloud-secretmanager",
					"google-cloud-secretmanager-bom",
					"", // parent
				}
				for _, d := range dirs {
					if err := os.MkdirAll(filepath.Join(libraryDir, d), 0755); err != nil {
						t.Fatal(err)
					}
				}
			},
			want: []javaModule{
				{artifactID: "proto-google-cloud-secretmanager-v1", isMissing: true, template: protoPOMTemplateName},
				{artifactID: "google-cloud-secretmanager", isMissing: true, template: clientPOMTemplateName},
				{artifactID: "google-cloud-secretmanager-bom", isMissing: true, template: bomPOMTemplateName},
				{artifactID: "google-cloud-secretmanager-parent", isMissing: true, template: parentPOMTemplateName},
			},
		},
		{
			name: "existing poms",
			library: &config.Library{
				Name:    "secretmanager",
				Version: "1.2.3",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			monorepoVersion: "1.2.3",
			metadata: &repoMetadata{
				NamePretty: "Secret Manager",
			},
			transports: map[string]serviceconfig.Transport{
				"google/cloud/secretmanager/v1": serviceconfig.GRPC,
			},
			setup: func(t *testing.T, libraryDir string) {
				dirs := []string{
					"proto-google-cloud-secretmanager-v1",
					"grpc-google-cloud-secretmanager-v1",
					"google-cloud-secretmanager",
					"google-cloud-secretmanager-bom",
					"", // parent
				}
				for _, d := range dirs {
					dir := filepath.Join(libraryDir, d)
					if err := os.MkdirAll(dir, 0755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(dir, "pom.xml"), []byte("<project/>"), 0644); err != nil {
						t.Fatal(err)
					}
				}
			},
			want: []javaModule{
				{artifactID: "proto-google-cloud-secretmanager-v1", isMissing: false, template: protoPOMTemplateName},
				{artifactID: "grpc-google-cloud-secretmanager-v1", isMissing: false, template: gRPCPOMTemplateName},
				{artifactID: "google-cloud-secretmanager", isMissing: false, template: clientPOMTemplateName},
				{artifactID: "google-cloud-secretmanager-bom", isMissing: false, template: bomPOMTemplateName},
				{artifactID: "google-cloud-secretmanager-parent", isMissing: false, template: parentPOMTemplateName},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			if test.setup != nil {
				test.setup(t, tmpDir)
			}
			got, err := collectModules(test.library, tmpDir, test.monorepoVersion, test.metadata, test.transports)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(
				test.want,
				got,
				cmp.AllowUnexported(javaModule{}),
				cmpopts.IgnoreFields(javaModule{}, "dir", "templateData")); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCollectModules_Error(t *testing.T) {

	for _, test := range []struct {
		name       string
		library    *config.Library
		transports map[string]serviceconfig.Transport
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
			transports: map[string]serviceconfig.Transport{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := collectModules(test.library, t.TempDir(), "1.2.3", &repoMetadata{}, test.transports); err == nil {
				t.Error("collectModules() error = nil, want non-nil")
			}
		})
	}
}

func TestIsPOMMissing(t *testing.T) {
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
			got, err := isPOMMissing(dir)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Errorf("isPOMMissing(%q) = %v, want %v", dir, got, test.want)
			}
		})
	}
}

func TestIsPOMMissing_DirMissingError(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent")
	_, err := isPOMMissing(dir)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("isPOMMissing(%q) error = %v, want %v", dir, err, fs.ErrNotExist)
	}
}
