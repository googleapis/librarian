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
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/testhelper"
	"github.com/googleapis/librarian/internal/yaml"
)

// maxParentTraversals defines the maximum directory levels to walk upwards when locating repositories.
const maxParentTraversals = 6

var hexSHARegexp = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)

func TestSecretManager_IntegrationPilot_DynamicResolution(t *testing.T) {
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
		return lib != nil && lib.Name == "google-cloud-secret-manager"
	})
	if idx == -1 {
		t.Fatal("google-cloud-secret-manager library entry not found in librarian.yaml")
	}
	secretManagerLib := cfg.Libraries[idx]

	want := &config.PythonPackage{
		PythonDefault: config.PythonDefault{
			Generator: "sidekick",
		},
		MetadataNameOverride: "secretmanager",
		DefaultVersion:       "v1",
	}
	if diff := cmp.Diff(want, secretManagerLib.Python); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestSecretManager_IntegrationPilot_ProtoSources(t *testing.T) {
	googleapisDir := findGoogleapisDir(t)

	for _, test := range []struct {
		name string
		path string
	}{
		{
			name: "service proto",
			path: "google/cloud/secretmanager/v1/service.proto",
		},
		{
			name: "resources proto",
			path: "google/cloud/secretmanager/v1/resources.proto",
		},
		{
			name: "service yaml",
			path: "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
		},
		{
			name: "grpc service config",
			path: "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json",
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

func TestSecretManager_IntegrationPilot_ModelCreation(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	googleapisDir := findGoogleapisDir(t)

	lib := &config.Library{
		Name:  "google-cloud-secret-manager",
		Roots: []string{"googleapis"},
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
		return svc != nil && svc.Name == "SecretManagerService"
	})
	if idx == -1 {
		t.Fatal("service SecretManagerService not found in model services")
	}
	svc := model.Services[idx]

	expectedMethods := []string{
		"ListSecrets",
		"CreateSecret",
		"AddSecretVersion",
		"GetSecret",
		"UpdateSecret",
		"DeleteSecret",
		"ListSecretVersions",
		"GetSecretVersion",
		"AccessSecretVersion",
		"DisableSecretVersion",
		"EnableSecretVersion",
		"DestroySecretVersion",
		"SetIamPolicy",
		"GetIamPolicy",
		"TestIamPermissions",
		"EnableManagedRotation",
		"RotateSecret",
		"ListLocations",
		"GetLocation",
	}
	var methodNames []string
	for _, m := range svc.Methods {
		methodNames = append(methodNames, m.Name)
	}

	if diff := cmp.Diff(expectedMethods, methodNames); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestSecretManager_IntegrationPilot_HarnessExecution(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	googleapisDir := findGoogleapisDir(t)

	if _, err := exec.LookPath("ruff"); err != nil {
		mockDir := t.TempDir()
		testhelper.WriteExecutable(t, filepath.Join(mockDir, "ruff"), "#!/bin/sh\nexit 0\n")
		t.Setenv("PATH", mockDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	}

	outDir := t.TempDir()
	lib := &config.Library{
		Name:   "google-cloud-secret-manager",
		Output: outDir,
		Roots:  []string{"googleapis"},
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

	for _, test := range []struct {
		name string
		path string
	}{
		{
			name: "package gapic version",
			path: "google/cloud/secretmanager/gapic_version.py",
		},
		{
			name: "package py.typed",
			path: "google/cloud/secretmanager/py.typed",
		},
		{
			name: "v1 compat",
			path: "google/cloud/secretmanager_v1/_compat.py",
		},
		{
			name: "v1 gapic version",
			path: "google/cloud/secretmanager_v1/gapic_version.py",
		},
		{
			name: "v1 py.typed",
			path: "google/cloud/secretmanager_v1/py.typed",
		},
		{
			name: "client",
			path: "google/cloud/secretmanager_v1/services/secret_manager_service/client.py",
		},
		{
			name: "async client",
			path: "google/cloud/secretmanager_v1/services/secret_manager_service/async_client.py",
		},
		{
			name: "transport base",
			path: "google/cloud/secretmanager_v1/services/secret_manager_service/transports/base.py",
		},
		{
			name: "transport grpc",
			path: "google/cloud/secretmanager_v1/services/secret_manager_service/transports/grpc.py",
		},
		{
			name: "transport grpc asyncio",
			path: "google/cloud/secretmanager_v1/services/secret_manager_service/transports/grpc_asyncio.py",
		},
		{
			name: "transport rest",
			path: "google/cloud/secretmanager_v1/services/secret_manager_service/transports/rest.py",
		},
		{
			name: "transport rest base",
			path: "google/cloud/secretmanager_v1/services/secret_manager_service/transports/rest_base.py",
		},
		{
			name: "types resources",
			path: "google/cloud/secretmanager_v1/types/resources.py",
		},
		{
			name: "types service",
			path: "google/cloud/secretmanager_v1/types/service.py",
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

func findGoogleapisDir(t *testing.T) string {
	t.Helper()
	if envDir := os.Getenv("GOOGLEAPIS_DIR"); envDir != "" {
		if abs, absErr := filepath.Abs(envDir); absErr == nil {
			if _, statErr := os.Stat(filepath.Join(abs, "google/cloud/secretmanager/v1/service.proto")); statErr == nil {
				return abs
			}
		}
	}
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for range maxParentTraversals {
		candidates := []string{
			filepath.Join(dir, "googleapis", "main"),
			filepath.Join(dir, "googleapis"),
			filepath.Join(dir, "migration-experiments", "googleapis", "main"),
			filepath.Join(dir, "migration-experiments", "googleapis"),
		}
		for _, c := range candidates {
			protoPath := filepath.Join(c, "google/cloud/secretmanager/v1/service.proto")
			if _, statErr := os.Stat(protoPath); statErr == nil {
				if abs, absErr := filepath.Abs(c); absErr == nil {
					return abs
				}
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Skip("skipping integration pilot test: googleapis directory not found")
	return ""
}

func findGoogleCloudPythonDir(t *testing.T) string {
	t.Helper()
	if envDir := os.Getenv("GOOGLE_CLOUD_PYTHON_DIR"); envDir != "" {
		if abs, absErr := filepath.Abs(envDir); absErr == nil {
			if _, statErr := os.Stat(filepath.Join(abs, "librarian.yaml")); statErr == nil {
				return abs
			}
		}
	}
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for range maxParentTraversals {
		candidates := []string{
			filepath.Join(dir, "google-cloud-python", "python-planning-02"),
			filepath.Join(dir, "google-cloud-python", "main"),
			filepath.Join(dir, "google-cloud-python"),
			filepath.Join(dir, "migration-experiments", "google-cloud-python", "python-planning-02"),
		}
		for _, c := range candidates {
			yamlPath := filepath.Join(c, "librarian.yaml")
			if _, statErr := os.Stat(yamlPath); statErr == nil {
				if abs, absErr := filepath.Abs(c); absErr == nil {
					return abs
				}
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func findGoogleCloudPythonConfig(t *testing.T) *config.Config {
	t.Helper()
	dir := findGoogleCloudPythonDir(t)
	if dir == "" {
		return nil
	}
	yamlPath := filepath.Join(dir, "librarian.yaml")
	cfg, err := yaml.Read[config.Config](yamlPath)
	if err != nil {
		t.Logf("warning: failed to read %s: %v", yamlPath, err)
		return nil
	}
	return cfg
}

func resolveGoogleapisCommitSHA(t *testing.T, googleapisDir string) string {
	t.Helper()
	if commit := os.Getenv("GOOGLEAPIS_COMMIT"); commit != "" {
		if hexSHARegexp.MatchString(commit) {
			return commit
		}
	}

	if cfg := findGoogleCloudPythonConfig(t); cfg != nil && cfg.Sources != nil && cfg.Sources.Googleapis != nil {
		commit := cfg.Sources.Googleapis.Commit
		if hexSHARegexp.MatchString(commit) {
			return commit
		}
	}

	cmd := exec.CommandContext(t.Context(), "git", "-C", googleapisDir, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err == nil {
		commit := strings.TrimSpace(string(out))
		if hexSHARegexp.MatchString(commit) {
			return commit
		}
	}

	t.Fatal("unable to dynamically resolve googleapis commit SHA")
	return ""
}
