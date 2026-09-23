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

func TestCompute_IntegrationPilot_DynamicResolution(t *testing.T) {
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
		return lib != nil && lib.Name == "google-cloud-compute"
	})
	if idx == -1 {
		t.Fatal("google-cloud-compute library entry not found in librarian.yaml")
	}
	computeLib := cfg.Libraries[idx]

	want := &config.PythonPackage{
		PythonDefault: config.PythonDefault{
			Generator: "sidekick",
		},
		MetadataNameOverride: "compute",
		DefaultVersion:       "v1",
	}
	if diff := cmp.Diff(want, computeLib.Python); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestCompute_IntegrationPilot_ProtoSources(t *testing.T) {
	googleapisDir := findGoogleapisDir(t)

	for _, test := range []struct {
		name string
		path string
	}{
		{
			name: "compute proto",
			path: "google/cloud/compute/v1/compute.proto",
		},
		{
			name: "compute yaml",
			path: "google/cloud/compute/v1/compute_v1.yaml",
		},
		{
			name: "grpc service config",
			path: "google/cloud/compute/v1/compute_grpc_service_config.json",
		},
		{
			name: "gapic yaml",
			path: "google/cloud/compute/v1/compute_gapic.yaml",
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

func TestCompute_IntegrationPilot_ModelCreation(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	googleapisDir := findGoogleapisDir(t)

	lib := &config.Library{
		Name:  "google-cloud-compute",
		Roots: []string{"googleapis"},
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

	if got, want := model.PackageName, "google.cloud.compute.v1"; got != want {
		t.Errorf("model.PackageName = %q, want %q", got, want)
	}

	var serviceNames []string
	for _, svc := range model.Services {
		serviceNames = append(serviceNames, svc.Name)
	}

	coreServices := []string{
		"Instances",
		"Addresses",
		"Disks",
		"AcceleratorTypes",
		"GlobalOperations",
		"RegionOperations",
		"ZoneOperations",
	}
	for _, wantSvc := range coreServices {
		if !slices.Contains(serviceNames, wantSvc) {
			t.Errorf("service %q not found in model services: %v", wantSvc, serviceNames)
		}
	}

	idx := slices.IndexFunc(model.Services, func(svc *api.Service) bool {
		return svc != nil && svc.Name == "Instances"
	})
	if idx == -1 {
		t.Fatal("service Instances not found in model services")
	}
	instancesSvc := model.Services[idx]

	var instancesMethods []string
	for _, m := range instancesSvc.Methods {
		instancesMethods = append(instancesMethods, m.Name)
	}

	expectedMethods := []string{
		"AggregatedList",
		"Insert",
		"Delete",
		"Get",
		"List",
		"Reset",
		"Start",
		"Stop",
	}
	for _, expectedMethod := range expectedMethods {
		if !slices.Contains(instancesMethods, expectedMethod) {
			t.Errorf("expected method %q missing from Instances service", expectedMethod)
		}
	}
}

func TestCompute_IntegrationPilot_HarnessExecution(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	googleapisDir := findGoogleapisDir(t)

	if _, err := exec.LookPath("ruff"); err != nil {
		mockDir := t.TempDir()
		testhelper.WriteExecutable(t, filepath.Join(mockDir, "ruff"), "#!/bin/sh\nexit 0\n")
		t.Setenv("PATH", mockDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	}

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

	for _, test := range []struct {
		name string
		path string
	}{
		{
			name: "package gapic version",
			path: "google/cloud/compute/gapic_version.py",
		},
		{
			name: "package py.typed",
			path: "google/cloud/compute/py.typed",
		},
		{
			name: "v1 compat",
			path: "google/cloud/compute_v1/_compat.py",
		},
		{
			name: "v1 gapic version",
			path: "google/cloud/compute_v1/gapic_version.py",
		},
		{
			name: "v1 py.typed",
			path: "google/cloud/compute_v1/py.typed",
		},
		{
			name: "services init",
			path: "google/cloud/compute_v1/services/__init__.py",
		},
		{
			name: "instances client",
			path: "google/cloud/compute_v1/services/instances/client.py",
		},
		{
			name: "instances pagers",
			path: "google/cloud/compute_v1/services/instances/pagers.py",
		},
		{
			name: "instances transport base",
			path: "google/cloud/compute_v1/services/instances/transports/base.py",
		},
		{
			name: "instances transport rest",
			path: "google/cloud/compute_v1/services/instances/transports/rest.py",
		},
		{
			name: "instances transport rest base",
			path: "google/cloud/compute_v1/services/instances/transports/rest_base.py",
		},
		{
			name: "types init",
			path: "google/cloud/compute_v1/types/__init__.py",
		},
		{
			name: "types compute",
			path: "google/cloud/compute_v1/types/compute.py",
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
