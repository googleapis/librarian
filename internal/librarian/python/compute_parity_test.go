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

func TestCompute_IntegrationPilot_ParityValidation(t *testing.T) {
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
		Name:    "google-cloud-compute",
		Version: "1.54.0",
		Output:  outDir,
		Roots:   []string{"googleapis"},
		APIs: []*config.API{
			{Path: "google/cloud/compute/v1"},
		},
		Python: &config.PythonPackage{
			PythonDefault: config.PythonDefault{
				Generator: "sidekick",
			},
			MetadataNameOverride: "compute",
			DefaultVersion:       "v1",
		},
	}

	srcs := &sources.Sources{Googleapis: googleapisDir}
	if err := Generate(t.Context(), nil, lib, srcs); err != nil {
		t.Fatal(err)
	}

	prodDir := filepath.Join(gcpDir, "packages", "google-cloud-compute")

	for _, test := range []struct {
		name    string
		relPath string
	}{
		{name: "package gapic_version", relPath: "google/cloud/compute/gapic_version.py"},
		{name: "package py.typed", relPath: "google/cloud/compute/py.typed"},
		{name: "v1 compat", relPath: "google/cloud/compute_v1/_compat.py"},
		{name: "v1 gapic_version", relPath: "google/cloud/compute_v1/gapic_version.py"},
		{name: "v1 py.typed", relPath: "google/cloud/compute_v1/py.typed"},
		{name: "services init", relPath: "google/cloud/compute_v1/services/__init__.py"},
		{name: "instances init", relPath: "google/cloud/compute_v1/services/instances/__init__.py"},
		{name: "instances client", relPath: "google/cloud/compute_v1/services/instances/client.py"},
		{name: "instances pagers", relPath: "google/cloud/compute_v1/services/instances/pagers.py"},
		{name: "instances transport base", relPath: "google/cloud/compute_v1/services/instances/transports/base.py"},
		{name: "instances transport init", relPath: "google/cloud/compute_v1/services/instances/transports/__init__.py"},
		{name: "instances transport readme", relPath: "google/cloud/compute_v1/services/instances/transports/README.rst"},
		{name: "instances transport rest base", relPath: "google/cloud/compute_v1/services/instances/transports/rest_base.py"},
		{name: "instances transport rest", relPath: "google/cloud/compute_v1/services/instances/transports/rest.py"},
		{name: "types init", relPath: "google/cloud/compute_v1/types/__init__.py"},
		{name: "types compute", relPath: "google/cloud/compute_v1/types/compute.py"},
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
