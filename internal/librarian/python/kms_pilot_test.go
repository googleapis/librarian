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

func TestKMS_IntegrationPilot_DynamicResolution(t *testing.T) {
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
		return lib != nil && lib.Name == "google-cloud-kms"
	})
	if idx == -1 {
		t.Fatal("google-cloud-kms library entry not found in librarian.yaml")
	}
	kmsLib := cfg.Libraries[idx]

	want := &config.PythonPackage{
		PythonDefault: config.PythonDefault{
			Generator: "sidekick",
		},
		MetadataNameOverride: "cloudkms",
		DefaultVersion:       "v1",
	}
	if diff := cmp.Diff(want, kmsLib.Python); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestKMS_IntegrationPilot_ProtoSources(t *testing.T) {
	googleapisDir := findGoogleapisDir(t)

	for _, test := range []struct {
		name string
		path string
	}{
		{
			name: "service proto",
			path: "google/cloud/kms/v1/service.proto",
		},
		{
			name: "resources proto",
			path: "google/cloud/kms/v1/resources.proto",
		},
		{
			name: "ekm_service proto",
			path: "google/cloud/kms/v1/ekm_service.proto",
		},
		{
			name: "hsm_management proto",
			path: "google/cloud/kms/v1/hsm_management.proto",
		},
		{
			name: "autokey proto",
			path: "google/cloud/kms/v1/autokey.proto",
		},
		{
			name: "autokey_admin proto",
			path: "google/cloud/kms/v1/autokey_admin.proto",
		},
		{
			name: "cloudkms yaml",
			path: "google/cloud/kms/v1/cloudkms_v1.yaml",
		},
		{
			name: "grpc service config",
			path: "google/cloud/kms/v1/cloudkms_grpc_service_config.json",
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

func TestKMS_IntegrationPilot_ModelCreation(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	googleapisDir := findGoogleapisDir(t)

	lib := &config.Library{
		Name:  "google-cloud-kms",
		Roots: []string{"googleapis"},
		APIs: []*config.API{
			{Path: "google/cloud/kms/v1"},
		},
		Python: &config.PythonPackage{
			PythonDefault: config.PythonDefault{
				Generator: "sidekick",
			},
			MetadataNameOverride: "cloudkms",
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

	var serviceNames []string
	for _, svc := range model.Services {
		serviceNames = append(serviceNames, svc.Name)
	}

	expectedServices := []string{
		"Autokey",
		"AutokeyAdmin",
		"EkmService",
		"HsmManagement",
		"KeyManagementService",
	}
	for _, wantSvc := range expectedServices {
		if !slices.Contains(serviceNames, wantSvc) {
			t.Errorf("service %q not found in model services: %v", wantSvc, serviceNames)
		}
	}

	idx := slices.IndexFunc(model.Services, func(svc *api.Service) bool {
		return svc != nil && svc.Name == "KeyManagementService"
	})
	if idx == -1 {
		t.Fatal("service KeyManagementService not found in model services")
	}
	kmsSvc := model.Services[idx]

	var kmsMethods []string
	for _, m := range kmsSvc.Methods {
		kmsMethods = append(kmsMethods, m.Name)
	}

	for _, expectedMethod := range []string{
		"ListKeyRings",
		"ListCryptoKeys",
		"ListCryptoKeyVersions",
		"ListImportJobs",
		"GetKeyRing",
		"GetCryptoKey",
		"GetCryptoKeyVersion",
		"GetPublicKey",
		"GetImportJob",
		"CreateKeyRing",
		"CreateCryptoKey",
		"CreateCryptoKeyVersion",
		"UpdateCryptoKey",
		"UpdateCryptoKeyVersion",
		"UpdateCryptoKeyPrimaryVersion",
		"DestroyCryptoKeyVersion",
		"RestoreCryptoKeyVersion",
		"CreateImportJob",
		"ImportCryptoKeyVersion",
		"Encrypt",
		"Decrypt",
		"AsymmetricSign",
		"AsymmetricDecrypt",
		"MacSign",
		"MacVerify",
		"GenerateRandomBytes",
		"SetIamPolicy",
		"GetIamPolicy",
		"TestIamPermissions",
		"ListLocations",
		"GetLocation",
		"GetOperation",
	} {
		if !slices.Contains(kmsMethods, expectedMethod) {
			t.Errorf("expected method %q missing from KeyManagementService", expectedMethod)
		}
	}
}

func TestKMS_IntegrationPilot_HarnessExecution(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	googleapisDir := findGoogleapisDir(t)

	if _, err := exec.LookPath("ruff"); err != nil {
		mockDir := t.TempDir()
		testhelper.WriteExecutable(t, filepath.Join(mockDir, "ruff"), "#!/bin/sh\nexit 0\n")
		t.Setenv("PATH", mockDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	}

	outDir := t.TempDir()
	lib := &config.Library{
		Name:    "google-cloud-kms",
		Version: "3.17.0",
		Output:  outDir,
		Roots:   []string{"googleapis"},
		APIs: []*config.API{
			{Path: "google/cloud/kms/v1"},
		},
		Python: &config.PythonPackage{
			PythonDefault: config.PythonDefault{
				Generator: "sidekick",
			},
			MetadataNameOverride: "cloudkms",
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
			name: "package gapic_version",
			path: "google/cloud/kms/gapic_version.py",
		},
		{
			name: "package py.typed",
			path: "google/cloud/kms/py.typed",
		},
		{
			name: "v1 compat",
			path: "google/cloud/kms_v1/_compat.py",
		},
		{
			name: "v1 gapic_version",
			path: "google/cloud/kms_v1/gapic_version.py",
		},
		{
			name: "v1 py.typed",
			path: "google/cloud/kms_v1/py.typed",
		},
		{
			name: "services init",
			path: "google/cloud/kms_v1/services/__init__.py",
		},
		{
			name: "key management service client",
			path: "google/cloud/kms_v1/services/key_management_service/client.py",
		},
		{
			name: "key management service async client",
			path: "google/cloud/kms_v1/services/key_management_service/async_client.py",
		},
		{
			name: "key management service pagers",
			path: "google/cloud/kms_v1/services/key_management_service/pagers.py",
		},
		{
			name: "key management service transport base",
			path: "google/cloud/kms_v1/services/key_management_service/transports/base.py",
		},
		{
			name: "key management service transport grpc",
			path: "google/cloud/kms_v1/services/key_management_service/transports/grpc.py",
		},
		{
			name: "key management service transport grpc asyncio",
			path: "google/cloud/kms_v1/services/key_management_service/transports/grpc_asyncio.py",
		},
		{
			name: "key management service transport rest",
			path: "google/cloud/kms_v1/services/key_management_service/transports/rest.py",
		},
		{
			name: "key management service transport rest base",
			path: "google/cloud/kms_v1/services/key_management_service/transports/rest_base.py",
		},
		{
			name: "autokey client",
			path: "google/cloud/kms_v1/services/autokey/client.py",
		},
		{
			name: "autokey async client",
			path: "google/cloud/kms_v1/services/autokey/async_client.py",
		},
		{
			name: "autokey admin client",
			path: "google/cloud/kms_v1/services/autokey_admin/client.py",
		},
		{
			name: "autokey admin async client",
			path: "google/cloud/kms_v1/services/autokey_admin/async_client.py",
		},
		{
			name: "ekm service client",
			path: "google/cloud/kms_v1/services/ekm_service/client.py",
		},
		{
			name: "ekm service async client",
			path: "google/cloud/kms_v1/services/ekm_service/async_client.py",
		},
		{
			name: "hsm management client",
			path: "google/cloud/kms_v1/services/hsm_management/client.py",
		},
		{
			name: "hsm management async client",
			path: "google/cloud/kms_v1/services/hsm_management/async_client.py",
		},
		{
			name: "types init",
			path: "google/cloud/kms_v1/types/__init__.py",
		},
		{
			name: "types service",
			path: "google/cloud/kms_v1/types/service.py",
		},
		{
			name: "types resources",
			path: "google/cloud/kms_v1/types/resources.py",
		},
		{
			name: "types autokey",
			path: "google/cloud/kms_v1/types/autokey.py",
		},
		{
			name: "types autokey admin",
			path: "google/cloud/kms_v1/types/autokey_admin.py",
		},
		{
			name: "types ekm service",
			path: "google/cloud/kms_v1/types/ekm_service.py",
		},
		{
			name: "types hsm management",
			path: "google/cloud/kms_v1/types/hsm_management.py",
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
