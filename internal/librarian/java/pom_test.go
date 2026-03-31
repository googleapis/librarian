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
	// Pre-create the directories that syncPoms expects to exist.
	protoArtifactID := "proto-google-cloud-secretmanager-v1"
	grpcArtifactID := "grpc-google-cloud-secretmanager-v1"
	if err := os.MkdirAll(filepath.Join(tmpDir, protoArtifactID), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, grpcArtifactID), 0755); err != nil {
		t.Fatal(err)
	}
	if err := generatePomsIfMissing(library, tmpDir, googleapisDir); err != nil {
		t.Fatal(err)
	}
	artifacts := []string{protoArtifactID, grpcArtifactID}
	for _, artifact := range artifacts {
		gotPath := filepath.Join(tmpDir, artifact, "pom.xml")
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
			if _, err := collectModules(test.library, t.TempDir(), "/nonexistent"); err == nil {
				t.Error("collectModules() error = nil, want non-nil")
			}
		})
	}
}

func TestIsPomMissing(t *testing.T) {
	for _, test := range []struct {
		name      string
		setup     func(t *testing.T) string
		want      bool
		wantErrIs error
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
		{
			name: "dir missing error",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent")
			},
			wantErrIs: os.ErrNotExist,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := test.setup(t)
			got, err := isPomMissing(dir)
			if !errors.Is(err, test.wantErrIs) {
				t.Fatalf("isPomMissing(%q) error = %v, want %v", dir, err, test.wantErrIs)
			}
			if got != test.want {
				t.Errorf("isPomMissing(%q) = %v, want %v", dir, got, test.want)
			}
		})
	}
}
