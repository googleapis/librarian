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

package clirr

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	tmpDir := t.TempDir()
	protoModulePath := filepath.Join(tmpDir, "proto-google-cloud-test-v1")
	srcDir := filepath.Join(protoModulePath, "src", "main", "java", "com", "google", "cloud", "test", "v1")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	orBuilderFile := filepath.Join(srcDir, "TestOrBuilder.java")
	if err := os.WriteFile(orBuilderFile, []byte("package com.google.cloud.test.v1; public interface TestOrBuilder {}"), 0644); err != nil {
		t.Fatalf("failed to write OrBuilder file: %v", err)
	}

	if err := Generate(protoModulePath); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	outputPath := filepath.Join(protoModulePath, "clirr-ignored-differences.xml")
	if _, err := os.Stat(outputPath); err != nil {
		t.Errorf("expected %s to exist: %v", outputPath, err)
	}
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}
	expected := "com/google/cloud/test/v1"
	if !strings.Contains(string(content), expected) {
		t.Errorf("expected generated file to contain %s, but got:\n%s", expected, string(content))
	}
	// Test generation skips if file already exists
	initialContent := "manual content"
	if err := os.WriteFile(outputPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("failed to write manual file: %v", err)
	}
	if err := Generate(protoModulePath); err != nil {
		t.Fatalf("Generate failed on existing file: %v", err)
	}
	newContent, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read file after second Generate: %v", err)
	}
	if string(newContent) != initialContent {
		t.Errorf("expected Generate to skip existing file, but content changed from %q to %q", initialContent, string(newContent))
	}
}

func TestGenerate_Error(t *testing.T) {
	if err := Generate(""); err == nil {
		t.Error("expected error for empty path, but got nil")
	}
}
