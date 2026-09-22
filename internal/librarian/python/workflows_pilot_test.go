//go:build integration

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

package python

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestWorkflows_IntegrationPilot_DynamicResolution(t *testing.T) {
	googleapisDir := findGoogleapisDir(t)
	commitSHA := resolveGoogleapisCommitSHA(t, googleapisDir)

	if !hexSHARegexp.MatchString(commitSHA) {
		t.Errorf("resolved commit SHA = %q, want 40-character hex string", commitSHA)
	}

	cfg := findGoogleCloudPythonConfig(t)
	if cfg == nil {
		t.Skip("skipping librarian.yaml validation: google-cloud-python directory not found")
	}

	idx := slices.IndexFunc(cfg.Libraries, func(lib *config.Library) bool {
		return lib != nil && lib.Name == "google-cloud-workflows"
	})
	if idx == -1 {
		t.Fatal("google-cloud-workflows library entry not found in librarian.yaml")
	}
	workflowsLib := cfg.Libraries[idx]

	want := &config.PythonPackage{
		PythonDefault: config.PythonDefault{
			Generator: "sidekick",
		},
		MetadataNameOverride: "workflows",
		DefaultVersion:       "v1",
	}
	if diff := cmp.Diff(want, workflowsLib.Python); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestWorkflows_IntegrationPilot_ProtoSources(t *testing.T) {
	googleapisDir := findGoogleapisDir(t)

	for _, test := range []struct {
		name string
		path string
	}{
		{
			name: "workflows proto",
			path: "google/cloud/workflows/v1/workflows.proto",
		},
		{
			name: "workflows yaml",
			path: "google/cloud/workflows/v1/workflows_v1.yaml",
		},
		{
			name: "grpc service config",
			path: "google/cloud/workflows/v1/workflows_grpc_service_config.json",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fullPath := filepath.Join(googleapisDir, test.path)
			info, err := os.Stat(fullPath)
			if err != nil {
				t.Fatal(err)
			}
			if info.Size() == 0 {
				t.Errorf("source file %s is unexpectedly empty", fullPath)
			}
		})
	}
}

func TestWorkflows_IntegrationPilot_ModelCreation(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	googleapisDir := findGoogleapisDir(t)

	lib := &config.Library{
		Name:  "google-cloud-workflows",
		Roots: []string{"googleapis"},
		APIs: []*config.API{
			{Path: "google/cloud/workflows/v1"},
		},
		Python: &config.PythonPackage{
			PythonDefault: config.PythonDefault{
				Generator: "sidekick",
			},
			MetadataNameOverride: "workflows",
			DefaultVersion:       "v1",
		},
	}

	srcs := &sources.Sources{Googleapis: googleapisDir}
	modelConfig, err := toModelConfig(lib, lib.APIs[0], srcs, nil)
	if err != nil {
		t.Fatal(err)
	}

	model, err := parser.CreateModel(modelConfig)
	if err != nil {
		t.Fatal(err)
	}
	if model == nil {
		t.Fatal("parser.CreateModel() returned nil model")
	}

	idx := slices.IndexFunc(model.Services, func(svc *api.Service) bool {
		return svc != nil && svc.Name == "Workflows"
	})
	if idx == -1 {
		t.Fatal("service Workflows not found in model services")
	}
	svc := model.Services[idx]

	var methodNames []string
	for _, m := range svc.Methods {
		methodNames = append(methodNames, m.Name)
	}

	expectedMethods := []string{
		"ListWorkflows",
		"GetWorkflow",
		"CreateWorkflow",
		"DeleteWorkflow",
		"UpdateWorkflow",
		"ListWorkflowRevisions",
		"ListLocations",
		"GetLocation",
		"ListOperations",
		"GetOperation",
		"DeleteOperation",
	}

	if diff := cmp.Diff(expectedMethods, methodNames); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestWorkflows_IntegrationPilot_HarnessExecution(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	googleapisDir := findGoogleapisDir(t)

	if _, err := exec.LookPath("ruff"); err != nil {
		mockDir := t.TempDir()
		testhelper.WriteExecutable(t, filepath.Join(mockDir, "ruff"), "#!/bin/sh\nexit 0\n")
		t.Setenv("PATH", mockDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	}

	outDir := t.TempDir()
	lib := &config.Library{
		Name:    "google-cloud-workflows",
		Version: "1.23.0",
		Output:  outDir,
		Roots:   []string{"googleapis"},
		APIs: []*config.API{
			{Path: "google/cloud/workflows/v1"},
		},
		Python: &config.PythonPackage{
			PythonDefault: config.PythonDefault{
				Generator: "sidekick",
			},
			MetadataNameOverride: "workflows",
			DefaultVersion:       "v1",
		},
	}

	srcs := &sources.Sources{Googleapis: googleapisDir}
	if err := Generate(t.Context(), nil, lib, srcs); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name string
		path string
	}{
		{
			name: "package gapic version",
			path: "google/cloud/workflows/gapic_version.py",
		},
		{
			name: "package py.typed",
			path: "google/cloud/workflows/py.typed",
		},
		{
			name: "v1 compat",
			path: "google/cloud/workflows_v1/_compat.py",
		},
		{
			name: "v1 gapic version",
			path: "google/cloud/workflows_v1/gapic_version.py",
		},
		{
			name: "v1 py.typed",
			path: "google/cloud/workflows_v1/py.typed",
		},
		{
			name: "services init",
			path: "google/cloud/workflows_v1/services/__init__.py",
		},
		{
			name: "service init",
			path: "google/cloud/workflows_v1/services/workflows/__init__.py",
		},
		{
			name: "client",
			path: "google/cloud/workflows_v1/services/workflows/client.py",
		},
		{
			name: "async client",
			path: "google/cloud/workflows_v1/services/workflows/async_client.py",
		},
		{
			name: "pagers",
			path: "google/cloud/workflows_v1/services/workflows/pagers.py",
		},
		{
			name: "transports readme",
			path: "google/cloud/workflows_v1/services/workflows/transports/README.rst",
		},
		{
			name: "transports init",
			path: "google/cloud/workflows_v1/services/workflows/transports/__init__.py",
		},
		{
			name: "transport base",
			path: "google/cloud/workflows_v1/services/workflows/transports/base.py",
		},
		{
			name: "transport grpc",
			path: "google/cloud/workflows_v1/services/workflows/transports/grpc.py",
		},
		{
			name: "transport grpc asyncio",
			path: "google/cloud/workflows_v1/services/workflows/transports/grpc_asyncio.py",
		},
		{
			name: "transport rest",
			path: "google/cloud/workflows_v1/services/workflows/transports/rest.py",
		},
		{
			name: "transport rest base",
			path: "google/cloud/workflows_v1/services/workflows/transports/rest_base.py",
		},
		{
			name: "types init",
			path: "google/cloud/workflows_v1/types/__init__.py",
		},
		{
			name: "types workflows",
			path: "google/cloud/workflows_v1/types/workflows.py",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fullPath := filepath.Join(outDir, test.path)
			info, err := os.Stat(fullPath)
			if err != nil {
				t.Fatal(err)
			}
			if info.Size() == 0 {
				t.Errorf("emitted file %s is unexpectedly empty", test.path)
			}
		})
	}
}
