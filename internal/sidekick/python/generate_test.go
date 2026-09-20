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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/language"
)

const wantTransportReadme = "\n" +
	"transport inheritance structure\n" +
	"_______________________________\n" +
	"\n" +
	"``SecretManagerServiceTransport`` is the ABC for all transports.\n" +
	"\n" +
	"- public child ``SecretManagerServiceGrpcTransport`` for sync gRPC transport (defined in ``grpc.py``).\n" +
	"- public child ``SecretManagerServiceGrpcAsyncIOTransport`` for async gRPC transport (defined in ``grpc_asyncio.py``).\n" +
	"- private child ``_BaseSecretManagerServiceRestTransport`` for base REST transport with inner classes ``_BaseMETHOD`` (defined in ``rest_base.py``).\n" +
	"- public child ``SecretManagerServiceRestTransport`` for sync REST transport with inner classes ``METHOD`` derived from the parent's corresponding ``_BaseMETHOD`` classes (defined in ``rest.py``).\n"

func TestGenerate(t *testing.T) {
	currentYear := fmt.Sprintf("%04d", time.Now().Year())

	for _, test := range []struct {
		name             string
		library          *config.Library
		wantVersion      string
		wantTyped        string
		wantServicesInit string
		wantServiceInit  string
	}{
		{
			name: "emits gapic_version.py and py.typed",
			library: &config.Library{
				Name:          "google-cloud-secretmanager",
				Version:       "2.1.0",
				CopyrightYear: "2026",
			},
			wantVersion: `# -*- coding: utf-8 -*-
# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
__version__ = "2.1.0"  # {x-release-please-version}
`,
			wantTyped: `# Marker file for PEP 561.
# The google-cloud-secretmanager package uses inline types.
`,
			wantServicesInit: `# -*- coding: utf-8 -*-
# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
`,
			wantServiceInit: `# -*- coding: utf-8 -*-
# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
from .client import SecretManagerServiceClient
from .async_client import SecretManagerServiceAsyncClient

__all__ = (
    'SecretManagerServiceClient',
    'SecretManagerServiceAsyncClient',
)
`,
		},
		{
			name:    "emits gapic_version.py and py.typed with nil library defaults",
			library: nil,
			wantVersion: fmt.Sprintf(`# -*- coding: utf-8 -*-
# Copyright %s Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
__version__ = "0.0.0"  # {x-release-please-version}
`, currentYear),
			wantTyped: `# Marker file for PEP 561.
# The google-cloud-secretmanager package uses inline types.
`,
			wantServicesInit: fmt.Sprintf(`# -*- coding: utf-8 -*-
# Copyright %s Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
`, currentYear),
			wantServiceInit: fmt.Sprintf(`# -*- coding: utf-8 -*-
# Copyright %s Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
from .client import SecretManagerServiceClient
from .async_client import SecretManagerServiceAsyncClient

__all__ = (
    'SecretManagerServiceClient',
    'SecretManagerServiceAsyncClient',
)
`, currentYear),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outdir := t.TempDir()

			msg := api.NewTestMessage("Secret").WithFields(
				api.NewTestField("name").WithType(api.TypezString),
			)
			svc := api.NewTestService("SecretManagerService")
			model := api.NewTestAPI([]*api.Message{msg}, nil, []*api.Service{svc}).
				WithPackageName("google.cloud.secretmanager.v1")
			model.Name = "google-cloud-secretmanager"

			if err := Generate(t.Context(), model, outdir, test.library); err != nil {
				t.Fatal(err)
			}

			// Version and typed files in package directory
			pkgDir := filepath.Join(outdir, "google", "cloud", "secretmanager_v1")
			versionFile := filepath.Join(pkgDir, "gapic_version.py")
			versionBytes, err := os.ReadFile(versionFile)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantVersion, string(versionBytes)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}

			typedFile := filepath.Join(pkgDir, "py.typed")
			typedBytes, err := os.ReadFile(typedFile)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantTyped, string(typedBytes)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}

			// Version and typed files in root package directory
			rootDir := filepath.Join(outdir, "google", "cloud", "secretmanager")
			rootVersionBytes, err := os.ReadFile(filepath.Join(rootDir, "gapic_version.py"))
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantVersion, string(rootVersionBytes)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}

			rootTypedBytes, err := os.ReadFile(filepath.Join(rootDir, "py.typed"))
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantTyped, string(rootTypedBytes)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}

			// services/__init__.py
			servicesInitBytes, err := os.ReadFile(filepath.Join(pkgDir, "services", "__init__.py"))
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantServicesInit, string(servicesInitBytes)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}

			// services/secret_manager_service/__init__.py
			serviceInitBytes, err := os.ReadFile(filepath.Join(pkgDir, "services", "secret_manager_service", "__init__.py"))
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantServiceInit, string(serviceInitBytes)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}

			// services/secret_manager_service/transports/README.rst
			readmeBytes, err := os.ReadFile(filepath.Join(pkgDir, "services", "secret_manager_service", "transports", "README.rst"))
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(wantTransportReadme, string(readmeBytes)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerate_Error(t *testing.T) {
	outdir := t.TempDir()
	err := Generate(t.Context(), nil, outdir, nil)
	if !errors.Is(err, ErrNilModel) {
		t.Errorf("Generate(ctx, nil, %q, nil) error = %v, want %v", outdir, err, ErrNilModel)
	}
}

func TestValidateOutputContainment(t *testing.T) {
	outdir := t.TempDir()
	files := []language.GeneratedFile{
		{OutputPath: "gapic_version.py"},
		{OutputPath: "google/cloud/secretmanager_v1/gapic_version.py"},
		{OutputPath: "google/cloud/secretmanager_v1/services/__init__.py"},
	}
	if err := validateOutputContainment(outdir, files); err != nil {
		t.Fatal(err)
	}
}

func TestValidateOutputContainment_Error(t *testing.T) {
	outdir := t.TempDir()
	for _, test := range []struct {
		name    string
		files   []language.GeneratedFile
		wantErr error
	}{
		{
			name: "empty output path",
			files: []language.GeneratedFile{
				{OutputPath: ""},
			},
			wantErr: ErrInvalidOutputPath,
		},
		{
			name: "absolute output path",
			files: []language.GeneratedFile{
				{OutputPath: "/tmp/secretmanager/gapic_version.py"},
			},
			wantErr: ErrInvalidOutputPath,
		},
		{
			name: "relative path with leading dot dot",
			files: []language.GeneratedFile{
				{OutputPath: "../gapic_version.py"},
			},
			wantErr: ErrEscapeOutputDir,
		},
		{
			name: "nested path escaping directory",
			files: []language.GeneratedFile{
				{OutputPath: "foo/../../gapic_version.py"},
			},
			wantErr: ErrEscapeOutputDir,
		},
		{
			name: "duplicate output path",
			files: []language.GeneratedFile{
				{OutputPath: "google/cloud/gapic_version.py"},
				{OutputPath: "google/cloud/gapic_version.py"},
			},
			wantErr: ErrDuplicateOutputPath,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := validateOutputContainment(outdir, test.files)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("validateOutputContainment error = %v, want %v", err, test.wantErr)
			}
		})
	}
}
