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

func TestKMS_IntegrationPilot_ParityValidation(t *testing.T) {
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

	prodDir := filepath.Join(gcpDir, "packages", "google-cloud-kms")

	for _, test := range []struct {
		name    string
		relPath string
	}{
		{name: "package gapic_version", relPath: "google/cloud/kms/gapic_version.py"},
		{name: "package py.typed", relPath: "google/cloud/kms/py.typed"},
		{name: "v1 compat", relPath: "google/cloud/kms_v1/_compat.py"},
		{name: "v1 gapic_version", relPath: "google/cloud/kms_v1/gapic_version.py"},
		{name: "v1 py.typed", relPath: "google/cloud/kms_v1/py.typed"},
		{name: "services init", relPath: "google/cloud/kms_v1/services/__init__.py"},
		{name: "autokey admin async client", relPath: "google/cloud/kms_v1/services/autokey_admin/async_client.py"},
		{name: "autokey admin client", relPath: "google/cloud/kms_v1/services/autokey_admin/client.py"},
		{name: "autokey admin init", relPath: "google/cloud/kms_v1/services/autokey_admin/__init__.py"},
		{name: "autokey admin transport base", relPath: "google/cloud/kms_v1/services/autokey_admin/transports/base.py"},
		{name: "autokey admin transport grpc asyncio", relPath: "google/cloud/kms_v1/services/autokey_admin/transports/grpc_asyncio.py"},
		{name: "autokey admin transport grpc", relPath: "google/cloud/kms_v1/services/autokey_admin/transports/grpc.py"},
		{name: "autokey admin transport init", relPath: "google/cloud/kms_v1/services/autokey_admin/transports/__init__.py"},
		{name: "autokey admin transport readme", relPath: "google/cloud/kms_v1/services/autokey_admin/transports/README.rst"},
		{name: "autokey admin transport rest base", relPath: "google/cloud/kms_v1/services/autokey_admin/transports/rest_base.py"},
		{name: "autokey admin transport rest", relPath: "google/cloud/kms_v1/services/autokey_admin/transports/rest.py"},
		{name: "autokey async client", relPath: "google/cloud/kms_v1/services/autokey/async_client.py"},
		{name: "autokey client", relPath: "google/cloud/kms_v1/services/autokey/client.py"},
		{name: "autokey init", relPath: "google/cloud/kms_v1/services/autokey/__init__.py"},
		{name: "autokey pagers", relPath: "google/cloud/kms_v1/services/autokey/pagers.py"},
		{name: "autokey transport base", relPath: "google/cloud/kms_v1/services/autokey/transports/base.py"},
		{name: "autokey transport grpc asyncio", relPath: "google/cloud/kms_v1/services/autokey/transports/grpc_asyncio.py"},
		{name: "autokey transport grpc", relPath: "google/cloud/kms_v1/services/autokey/transports/grpc.py"},
		{name: "autokey transport init", relPath: "google/cloud/kms_v1/services/autokey/transports/__init__.py"},
		{name: "autokey transport readme", relPath: "google/cloud/kms_v1/services/autokey/transports/README.rst"},
		{name: "autokey transport rest base", relPath: "google/cloud/kms_v1/services/autokey/transports/rest_base.py"},
		{name: "autokey transport rest", relPath: "google/cloud/kms_v1/services/autokey/transports/rest.py"},
		{name: "ekm service async client", relPath: "google/cloud/kms_v1/services/ekm_service/async_client.py"},
		{name: "ekm service client", relPath: "google/cloud/kms_v1/services/ekm_service/client.py"},
		{name: "ekm service init", relPath: "google/cloud/kms_v1/services/ekm_service/__init__.py"},
		{name: "ekm service pagers", relPath: "google/cloud/kms_v1/services/ekm_service/pagers.py"},
		{name: "ekm service transport base", relPath: "google/cloud/kms_v1/services/ekm_service/transports/base.py"},
		{name: "ekm service transport grpc asyncio", relPath: "google/cloud/kms_v1/services/ekm_service/transports/grpc_asyncio.py"},
		{name: "ekm service transport grpc", relPath: "google/cloud/kms_v1/services/ekm_service/transports/grpc.py"},
		{name: "ekm service transport init", relPath: "google/cloud/kms_v1/services/ekm_service/transports/__init__.py"},
		{name: "ekm service transport readme", relPath: "google/cloud/kms_v1/services/ekm_service/transports/README.rst"},
		{name: "ekm service transport rest base", relPath: "google/cloud/kms_v1/services/ekm_service/transports/rest_base.py"},
		{name: "ekm service transport rest", relPath: "google/cloud/kms_v1/services/ekm_service/transports/rest.py"},
		{name: "hsm management async client", relPath: "google/cloud/kms_v1/services/hsm_management/async_client.py"},
		{name: "hsm management client", relPath: "google/cloud/kms_v1/services/hsm_management/client.py"},
		{name: "hsm management init", relPath: "google/cloud/kms_v1/services/hsm_management/__init__.py"},
		{name: "hsm management pagers", relPath: "google/cloud/kms_v1/services/hsm_management/pagers.py"},
		{name: "hsm management transport base", relPath: "google/cloud/kms_v1/services/hsm_management/transports/base.py"},
		{name: "hsm management transport grpc asyncio", relPath: "google/cloud/kms_v1/services/hsm_management/transports/grpc_asyncio.py"},
		{name: "hsm management transport grpc", relPath: "google/cloud/kms_v1/services/hsm_management/transports/grpc.py"},
		{name: "hsm management transport init", relPath: "google/cloud/kms_v1/services/hsm_management/transports/__init__.py"},
		{name: "hsm management transport readme", relPath: "google/cloud/kms_v1/services/hsm_management/transports/README.rst"},
		{name: "hsm management transport rest base", relPath: "google/cloud/kms_v1/services/hsm_management/transports/rest_base.py"},
		{name: "hsm management transport rest", relPath: "google/cloud/kms_v1/services/hsm_management/transports/rest.py"},
		{name: "key management service async client", relPath: "google/cloud/kms_v1/services/key_management_service/async_client.py"},
		{name: "key management service client", relPath: "google/cloud/kms_v1/services/key_management_service/client.py"},
		{name: "key management service init", relPath: "google/cloud/kms_v1/services/key_management_service/__init__.py"},
		{name: "key management service pagers", relPath: "google/cloud/kms_v1/services/key_management_service/pagers.py"},
		{name: "key management service transport base", relPath: "google/cloud/kms_v1/services/key_management_service/transports/base.py"},
		{name: "key management service transport grpc asyncio", relPath: "google/cloud/kms_v1/services/key_management_service/transports/grpc_asyncio.py"},
		{name: "key management service transport grpc", relPath: "google/cloud/kms_v1/services/key_management_service/transports/grpc.py"},
		{name: "key management service transport init", relPath: "google/cloud/kms_v1/services/key_management_service/transports/__init__.py"},
		{name: "key management service transport readme", relPath: "google/cloud/kms_v1/services/key_management_service/transports/README.rst"},
		{name: "key management service transport rest base", relPath: "google/cloud/kms_v1/services/key_management_service/transports/rest_base.py"},
		{name: "key management service transport rest", relPath: "google/cloud/kms_v1/services/key_management_service/transports/rest.py"},
		{name: "types autokey admin", relPath: "google/cloud/kms_v1/types/autokey_admin.py"},
		{name: "types autokey", relPath: "google/cloud/kms_v1/types/autokey.py"},
		{name: "types ekm service", relPath: "google/cloud/kms_v1/types/ekm_service.py"},
		{name: "types hsm management", relPath: "google/cloud/kms_v1/types/hsm_management.py"},
		{name: "types init", relPath: "google/cloud/kms_v1/types/__init__.py"},
		{name: "types resources", relPath: "google/cloud/kms_v1/types/resources.py"},
		{name: "types service", relPath: "google/cloud/kms_v1/types/service.py"},
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
