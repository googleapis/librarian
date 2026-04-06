// Copyright 2025 Google LLC
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

package surfer

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_Success(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
	}{
		{
			name: "valid command",
			args: []string{
				"surfer",
				"generate",
				"../gcloud/testdata/parallelstore/gcloud.yaml",
				"--googleapis", "../gcloud/testdata/googleapis",
				"--out", "../gcloud/testdata/parallelstore/surface",
			},
		},
		{
			name: "valid command with service-config",
			args: []string{
				"surfer",
				"generate",
				"../gcloud/testdata/parallelstore/gcloud.yaml",
				"--googleapis", "../gcloud/testdata/googleapis",
				"--out", "../gcloud/testdata/parallelstore/surface",
				"--service-config", "../gcloud/testdata/googleapis/google/cloud/parallelstore/v1/parallelstore_service.yaml",
			},
		},
		{
			name: "valid command with base-module",
			args: []string{
				"surfer",
				"generate",
				"../gcloud/testdata/parallelstore/gcloud.yaml",
				"--googleapis", "../gcloud/testdata/googleapis",
				"--out", "../gcloud/testdata/parallelstore/surface",
				"--base-module", "customsdk",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := Run(t.Context(), test.args...); err != nil {
				if strings.Contains(err.Error(), "failed to create API model") {
					return
				}
				t.Fatal(err)
			}
		})
	}
}

func TestRun_Errors(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
	}{
		{
			name: "invalid gcloud.yaml filepath",
			args: []string{
				"surfer",
				"generate",
				"invalidpath/gcloud.yaml",
				"--googleapis", "../gcloud/testdata/googleapis",
			},
		},
		{
			name: "missing config arg",
			args: []string{"surfer", "generate", "--googleapis", "../gcloud/testdata/googleapis"},
		},
		{
			name: "missing googleapis flag",
			args: []string{"surfer", "generate", "../gcloud/testdata/parallelstore/gcloud.yaml"},
		},
		{
			name: "missing descriptor-files-to-generate",
			args: []string{
				"surfer", "generate", "../gcloud/testdata/parallelstore/gcloud.yaml",
				"--descriptor-files", "dummy.desc",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := Run(t.Context(), test.args...); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestRun_Descriptors(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("skipping test because protoc is not installed")
	}

	coreGoogleapisPath := requireGoogleapisPath(t)
	scenarioDir := "testdata/field_attributes/input"

	tmpDir := t.TempDir()
	descFile := filepath.Join(tmpDir, "field_attributes.desc")

	cmd := exec.CommandContext(t.Context(), "protoc", "-o", descFile, "--include_imports",
		"-I", scenarioDir,
		"-I", coreGoogleapisPath,
		filepath.Join(scenarioDir, "field_attributes.proto"))
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to generate .desc file: %v", err)
	}

	args := []string{
		"surfer", "generate", filepath.Join(scenarioDir, "gcloud.yaml"),
		"--service-config", filepath.Join(scenarioDir, "service.yaml"),
		"--descriptor-files", descFile,
		"--descriptor-files-to-generate", "field_attributes.proto",
		"--out", filepath.Join(tmpDir, "out"),
	}

	if err := Run(t.Context(), args...); err != nil {
		t.Fatalf("surfer generate with descriptors failed: %v", err)
	}
}
