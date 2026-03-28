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
			t.Errorf("mismatch in %s (-want +got):\n%s", artifact, diff)
		}
	}
}

func TestWritePomIfMissing_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	data := grpcProtoPomData{
		ProtoArtifactID: "proto-artifact",
	}
	dir := filepath.Join(tmpDir, "exists")
	want := "existing content"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	pomPath := filepath.Join(dir, "pom.xml")
	if err := os.WriteFile(pomPath, []byte(want), 0644); err != nil {
		t.Fatal(err)
	}

	if err := writePomIfMissing(dir, protoPomTemplateName, data); err != nil {
		t.Errorf("writePomIfMissing(%q) error = %v, want nil", dir, err)
	}

	got, err := os.ReadFile(pomPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("got content %q, want %q", string(got), want)
	}
}

func TestWritePomIfMissing_Error(t *testing.T) {
	tmpDir := t.TempDir()
	data := grpcProtoPomData{
		ProtoArtifactID: "proto-artifact",
	}
	dir := filepath.Join(tmpDir, "nonexistent")
	err := writePomIfMissing(dir, protoPomTemplateName, data)
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("writePomIfMissing(%q) error = %v, wantErr %v", dir, err, os.ErrNotExist)
	}
}
