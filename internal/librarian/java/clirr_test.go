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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateClirrIgnore(t *testing.T) {
	tmpDir := t.TempDir()
	protoModulePath := filepath.Join(tmpDir, "proto-google-cloud-test-v1")
	srcDir := filepath.Join(protoModulePath, "src", "main", "java", "com", "google", "cloud", "test", "v1")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	orBuilderFile := filepath.Join(srcDir, "TestOrBuilder.java")
	if err := os.WriteFile(orBuilderFile, []byte("package com.google.cloud.test.v1; public interface TestOrBuilder {}"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := generateClirrIgnore(protoModulePath); err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(protoModulePath, "clirr-ignored-differences.xml")
	if _, err := os.Stat(outputPath); err != nil {
		t.Errorf("expected %s to exist: %v", outputPath, err)
	}
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	expected := "com/google/cloud/test/v1"
	if !strings.Contains(string(content), expected) {
		t.Errorf("expected generated file to contain %s, but got:\n%s", expected, string(content))
	}
}

func TestClirrIgnoreShouldGenerate(t *testing.T) {
	for _, test := range []struct {
		name       string
		artifactID string
		setup      func(t *testing.T, dir string)
		want       bool
	}{
		{
			name:       "should generate - prefix matches and not exists",
			artifactID: "proto-google-cloud-test-v1",
			setup:      func(t *testing.T, dir string) {},
			want:       true,
		},
		{
			name:       "should not generate - prefix mismatch",
			artifactID: "proto-data-manager-v1",
			setup:      func(t *testing.T, dir string) {},
			want:       false,
		},
		{
			name:       "should not generate - already exists",
			artifactID: "proto-google-cloud-test-v1",
			setup: func(t *testing.T, dir string) {
				path := filepath.Join(dir, clirrIgnoreFile)
				if err := os.WriteFile(path, []byte("exists"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			want: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			test.setup(t, dir)
			got, err := clirrIgnoreShouldGenerate(test.artifactID, dir)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Errorf("clirrIgnoreShouldGenerate(%q, %q) = %v, want %v", test.artifactID, dir, got, test.want)
			}
		})
	}
}
