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
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestWorkflows_IntegrationPilot_ParityValidation(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	googleapisDir := findGoogleapisDir(t)
	gcpDir := findGoogleCloudPythonDir(t)

	ruffDir := findRuffDir(t)
	if ruffDir == "" {
		mockDir := t.TempDir()
		testhelper.WriteExecutable(t, filepath.Join(mockDir, "ruff"), "#!/bin/sh\nexit 0\n")
		ruffDir = mockDir
	}
	t.Setenv("PATH", ruffDir+string(os.PathListSeparator)+os.Getenv("PATH"))

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

	prodDir := filepath.Join(gcpDir, "packages", "google-cloud-workflows")

	for _, test := range []struct {
		name    string
		relPath string
	}{
		{
			name:    "package gapic_version",
			relPath: "google/cloud/workflows/gapic_version.py",
		},
		{
			name:    "package py.typed",
			relPath: "google/cloud/workflows/py.typed",
		},
		{
			name:    "v1 compat",
			relPath: "google/cloud/workflows_v1/_compat.py",
		},
		{
			name:    "v1 gapic_version",
			relPath: "google/cloud/workflows_v1/gapic_version.py",
		},
		{
			name:    "v1 py.typed",
			relPath: "google/cloud/workflows_v1/py.typed",
		},
		{
			name:    "services init",
			relPath: "google/cloud/workflows_v1/services/__init__.py",
		},
		{
			name:    "service init",
			relPath: "google/cloud/workflows_v1/services/workflows/__init__.py",
		},
		{
			name:    "client",
			relPath: "google/cloud/workflows_v1/services/workflows/client.py",
		},
		{
			name:    "async client",
			relPath: "google/cloud/workflows_v1/services/workflows/async_client.py",
		},
		{
			name:    "pagers",
			relPath: "google/cloud/workflows_v1/services/workflows/pagers.py",
		},
		{
			name:    "transports readme",
			relPath: "google/cloud/workflows_v1/services/workflows/transports/README.rst",
		},
		{
			name:    "transports init",
			relPath: "google/cloud/workflows_v1/services/workflows/transports/__init__.py",
		},
		{
			name:    "transport base",
			relPath: "google/cloud/workflows_v1/services/workflows/transports/base.py",
		},
		{
			name:    "transport grpc",
			relPath: "google/cloud/workflows_v1/services/workflows/transports/grpc.py",
		},
		{
			name:    "transport grpc asyncio",
			relPath: "google/cloud/workflows_v1/services/workflows/transports/grpc_asyncio.py",
		},
		{
			name:    "transport rest base",
			relPath: "google/cloud/workflows_v1/services/workflows/transports/rest_base.py",
		},
		{
			name:    "transport rest",
			relPath: "google/cloud/workflows_v1/services/workflows/transports/rest.py",
		},
		{
			name:    "types init",
			relPath: "google/cloud/workflows_v1/types/__init__.py",
		},
		{
			name:    "types workflows",
			relPath: "google/cloud/workflows_v1/types/workflows.py",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotBytes, err := os.ReadFile(filepath.Join(outDir, test.relPath))
			if err != nil {
				t.Fatalf("failed to read generated file %q: %v", test.relPath, err)
			}
			wantBytes, err := os.ReadFile(filepath.Join(prodDir, test.relPath))
			if err != nil {
				t.Fatalf("failed to read production file %q: %v", test.relPath, err)
			}

			got := normalizeContent(string(gotBytes))
			want := normalizeContent(string(wantBytes))

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
