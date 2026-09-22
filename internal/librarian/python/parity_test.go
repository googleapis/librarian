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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestSecretManager_IntegrationPilot_ParityValidation(t *testing.T) {
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
		Name:    "google-cloud-secret-manager",
		Version: "2.30.0",
		Output:  outDir,
		Roots:   []string{"googleapis"},
		APIs: []*config.API{
			{Path: "google/cloud/secretmanager/v1"},
		},
		Python: &config.PythonPackage{
			PythonDefault: config.PythonDefault{
				Generator: "sidekick",
			},
			MetadataNameOverride: "secretmanager",
			DefaultVersion:       "v1",
		},
	}

	srcs := &sources.Sources{Googleapis: googleapisDir}
	if err := Generate(t.Context(), nil, lib, srcs); err != nil {
		t.Fatal(err)
	}

	prodDir := filepath.Join(gcpDir, "packages", "google-cloud-secret-manager")

	for _, test := range []struct {
		name    string
		relPath string
	}{
		{
			name:    "package gapic_version",
			relPath: "google/cloud/secretmanager/gapic_version.py",
		},
		{
			name:    "package py.typed",
			relPath: "google/cloud/secretmanager/py.typed",
		},
		{
			name:    "v1 compat",
			relPath: "google/cloud/secretmanager_v1/_compat.py",
		},
		{
			name:    "v1 gapic_version",
			relPath: "google/cloud/secretmanager_v1/gapic_version.py",
		},
		{
			name:    "v1 py.typed",
			relPath: "google/cloud/secretmanager_v1/py.typed",
		},
		{
			name:    "services init",
			relPath: "google/cloud/secretmanager_v1/services/__init__.py",
		},
		{
			name:    "service init",
			relPath: "google/cloud/secretmanager_v1/services/secret_manager_service/__init__.py",
		},
		{
			name:    "client",
			relPath: "google/cloud/secretmanager_v1/services/secret_manager_service/client.py",
		},
		{
			name:    "async client",
			relPath: "google/cloud/secretmanager_v1/services/secret_manager_service/async_client.py",
		},
		{
			name:    "pagers",
			relPath: "google/cloud/secretmanager_v1/services/secret_manager_service/pagers.py",
		},
		{
			name:    "transports readme",
			relPath: "google/cloud/secretmanager_v1/services/secret_manager_service/transports/README.rst",
		},
		{
			name:    "transports init",
			relPath: "google/cloud/secretmanager_v1/services/secret_manager_service/transports/__init__.py",
		},
		{
			name:    "transport base",
			relPath: "google/cloud/secretmanager_v1/services/secret_manager_service/transports/base.py",
		},
		{
			name:    "transport grpc",
			relPath: "google/cloud/secretmanager_v1/services/secret_manager_service/transports/grpc.py",
		},
		{
			name:    "transport grpc asyncio",
			relPath: "google/cloud/secretmanager_v1/services/secret_manager_service/transports/grpc_asyncio.py",
		},
		{
			name:    "transport rest base",
			relPath: "google/cloud/secretmanager_v1/services/secret_manager_service/transports/rest_base.py",
		},
		{
			name:    "transport rest",
			relPath: "google/cloud/secretmanager_v1/services/secret_manager_service/transports/rest.py",
		},
		{
			name:    "types init",
			relPath: "google/cloud/secretmanager_v1/types/__init__.py",
		},
		{
			name:    "types resources",
			relPath: "google/cloud/secretmanager_v1/types/resources.py",
		},
		{
			name:    "types service",
			relPath: "google/cloud/secretmanager_v1/types/service.py",
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

func normalizeContent(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

func findRuffDir(t *testing.T) string {
	t.Helper()
	if p, err := exec.LookPath("ruff"); err == nil {
		return filepath.Dir(p)
	}
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for range maxParentTraversals {
		candidate := filepath.Join(dir, ".venv", "bin", "ruff")
		if _, err := os.Stat(candidate); err == nil {
			return filepath.Dir(candidate)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
