// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package docker

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestNewEnvironmentProvider(t *testing.T) {
	const (
		testWorkRoot       = "testWorkRoot"
		testSecretsProject = "testSecretsProject"
	)
	pipelineConfig := &config.PipelineConfig{}
	ep := newEnvironmentProvider(testWorkRoot, testSecretsProject, pipelineConfig)
	if ep == nil {
		t.Fatal("newEnvironmentProvider() returned nil")
	}
	wantTmpFile := filepath.Join(testWorkRoot, "docker-env.txt")
	if ep.tmpFile != wantTmpFile {
		t.Errorf("ep.tmpFile = %q, want %q", ep.tmpFile, wantTmpFile)
	}
	if ep.secretsProject != testSecretsProject {
		t.Errorf("ep.secretsProject = %q, want %q", ep.secretsProject, testSecretsProject)
	}
	if ep.pipelineConfig != pipelineConfig {
		t.Error("ep.pipelineConfig is not the same as the one passed in")
	}
	if ep.secretCache == nil {
		t.Error("ep.secretCache is nil")
	}
}

func TestWriteEnvironmentFile(t *testing.T) {
	ctx := context.Background()
	testContent := "foo=bar\n"
	tmpDir := t.TempDir()
	e := newEnvironmentProvider(tmpDir, "", &config.PipelineConfig{
		Commands: map[string]*config.CommandConfig{
			"test-command": {
				EnvironmentVariables: []*config.CommandEnvironmentVariable{
					{Name: "foo", DefaultValue: "bar"},
				},
			},
		},
	})
	if err := e.writeEnvironmentFile(ctx, "test-command"); err != nil {
		t.Fatalf("writeEnvironmentFile() error = %v", err)
	}
	got, err := os.ReadFile(filepath.Join(tmpDir, "docker-env.txt"))
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(got) != testContent {
		t.Errorf("file content = %q, want %q", string(got), testContent)
	}
}

func TestConstructEnvironmentFileContent(t *testing.T) {
	const (
		testCommand     = "test-command"
		testHostVar     = "HOST_VAR"
		testHostValue   = "host_value"
		testSecretVar   = "SECRET_VAR"
		testSecretName  = "secret-name"
		testSecretValue = "secret_value"
	)
	pipelineConfig := &config.PipelineConfig{
		Commands: map[string]*config.CommandConfig{
			testCommand: {
				EnvironmentVariables: []*config.CommandEnvironmentVariable{
					{Name: testHostVar},
					{Name: testSecretVar, SecretName: testSecretName},
					{Name: "DEFAULT_VAR", DefaultValue: "default_value"},
					{Name: "UNRESOLVED_VAR"},
				},
			},
		},
	}

	e := newEnvironmentProvider("", "", pipelineConfig)
	e.secretCache[testSecretName] = testSecretValue
	t.Setenv(testHostVar, testHostValue)

	want := "HOST_VAR=host_value\nSECRET_VAR=secret_value\nDEFAULT_VAR=default_value\n# No value for UNRESOLVED_VAR\n"
	got, err := e.constructEnvironmentFileContent(context.Background(), testCommand)
	if err != nil {
		t.Fatalf("constructEnvironmentFileContent() error = %v", err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("constructEnvironmentFileContent() mismatch (-want +got):\n%s", diff)
	}
}
