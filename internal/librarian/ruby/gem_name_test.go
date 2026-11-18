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

package ruby

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestDefaultGemNames(t *testing.T) {
	for _, test := range []struct {
		name          string
		files         map[string]string
		apiPath       string
		wantWrapper   string
		wantVersioned string
	}{
		{
			name: "net-new API with camelCase segment",
			files: map[string]string{
				"google/cloud/workloadidentity/v1/service.proto": `syntax = "proto3";
option ruby_package = "Google::Cloud::WorkloadIdentity::V1";
`,
			},
			apiPath:       "google/cloud/workloadidentity/v1",
			wantWrapper:   "google-cloud-workload_identity",
			wantVersioned: "google-cloud-workload_identity-v1",
		},
		{
			name: "commented-out ruby_package line is ignored",
			files: map[string]string{
				"google/cloud/alloydb/v1beta/service.proto": `syntax = "proto3";
// option ruby_package = "Wrong::Comment::V1";
option ruby_package = "Google::Cloud::AlloyDB::V1beta";
`,
			},
			apiPath:       "google/cloud/alloydb/v1beta",
			wantWrapper:   "google-cloud-alloy_db",
			wantVersioned: "google-cloud-alloy_db-v1beta",
		},
		{
			name: "acronyms and deeply nested namespaces",
			files: map[string]string{
				"google/cloud/gdchardwaremanagement/v1alpha/service.proto": `syntax = "proto3";
option ruby_package = "Google::Cloud::GDCHardwareManagement::V1alpha";
`,
			},
			apiPath:       "google/cloud/gdchardwaremanagement/v1alpha",
			wantWrapper:   "google-cloud-gdc_hardware_management",
			wantVersioned: "google-cloud-gdc_hardware_management-v1alpha",
		},
		{
			name: "subdirectories with distinct sub-namespaces do not conflict with top-level protos",
			files: map[string]string{
				"google/cloud/gkehub/v1/service.proto": `syntax = "proto3";
option ruby_package = "Google::Cloud::GkeHub::V1";
`,
				"google/cloud/gkehub/v1/configmanagement/config.proto": `syntax = "proto3";
option ruby_package = "Google::Cloud::GkeHub::ConfigManagement::V1";
`,
			},
			apiPath:       "google/cloud/gkehub/v1",
			wantWrapper:   "google-cloud-gke_hub",
			wantVersioned: "google-cloud-gke_hub-v1",
		},
		{
			name: "two-segment ruby_package",
			files: map[string]string{
				"grafeas/v1/grafeas.proto": `syntax = "proto3";
option ruby_package = "Grafeas::V1";
`,
			},
			apiPath:       "grafeas/v1",
			wantWrapper:   "grafeas",
			wantVersioned: "grafeas-v1",
		},
		{
			name: "nested child service",
			files: map[string]string{
				"google/cloud/dialogflow/cx/v3/service.proto": `syntax = "proto3";
option ruby_package = "Google::Cloud::Dialogflow::CX::V3";
`,
			},
			apiPath:       "google/cloud/dialogflow/cx/v3",
			wantWrapper:   "google-cloud-dialogflow-cx",
			wantVersioned: "google-cloud-dialogflow-cx-v3",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			googleapisDir := t.TempDir()
			writeProtoFiles(t, googleapisDir, test.files)

			gotWrapper, gotVersioned, err := DefaultGemNames(googleapisDir, test.apiPath)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantWrapper, gotWrapper); diff != "" {
				t.Errorf("wrapper mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantVersioned, gotVersioned); diff != "" {
				t.Errorf("versioned mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDefaultGemNamesErrors(t *testing.T) {
	for _, test := range []struct {
		name    string
		files   map[string]string
		apiPath string
		wantErr error
	}{
		{
			name:    "missing directory",
			apiPath: "google/cloud/missing/v1",
			wantErr: os.ErrNotExist,
		},
		{
			name: "unversioned api path",
			files: map[string]string{
				"google/cloud/workloadidentity/service.proto": `syntax = "proto3";
option ruby_package = "Google::Cloud::WorkloadIdentity::V1";
`,
			},
			apiPath: "google/cloud/workloadidentity",
			wantErr: errRubyPackageVersionMismatch,
		},
		{
			name: "no proto files in directory",
			files: map[string]string{
				"google/cloud/empty/v1/README.md": "hello",
			},
			apiPath: "google/cloud/empty/v1",
			wantErr: os.ErrNotExist,
		},
		{
			name: "missing ruby_package in first proto file",
			files: map[string]string{
				"google/cloud/foo/v1/service.proto": `syntax = "proto3";
option ruby_package = "Google::Cloud::Foo::V1";
`,
				"google/cloud/foo/v1/resources.proto": `syntax = "proto3";
package google.cloud.foo.v1;
`,
			},
			apiPath: "google/cloud/foo/v1",
			wantErr: errMissingRubyPackage,
		},
		{
			name: "commented ruby_package counts as missing",
			files: map[string]string{
				"google/cloud/foo/v1/service.proto": `syntax = "proto3";
// option ruby_package = "Google::Cloud::Foo::V1";
`,
			},
			apiPath: "google/cloud/foo/v1",
			wantErr: errMissingRubyPackage,
		},
		{
			name: "empty ruby_package value",
			files: map[string]string{
				"google/cloud/foo/v1/service.proto": `syntax = "proto3";
option ruby_package = "";
`,
			},
			apiPath: "google/cloud/foo/v1",
			wantErr: errMissingRubyPackage,
		},
		{
			name: "single colon instead of double colon",
			files: map[string]string{
				"google/cloud/foo/v1/service.proto": `syntax = "proto3";
option ruby_package = "Google::Cloud:Foo::V1";
`,
			},
			apiPath: "google/cloud/foo/v1",
			wantErr: errInvalidRubyPackage,
		},
		{
			name: "lowercase segment",
			files: map[string]string{
				"google/cloud/foo/v1/service.proto": `syntax = "proto3";
option ruby_package = "google::Cloud::Foo::V1";
`,
			},
			apiPath: "google/cloud/foo/v1",
			wantErr: errInvalidRubyPackage,
		},
		{
			name: "single segment ruby_package",
			files: map[string]string{
				"google/cloud/foo/v1/service.proto": `syntax = "proto3";
option ruby_package = "V1";
`,
			},
			apiPath: "google/cloud/foo/v1",
			wantErr: errInvalidRubyPackage,
		},
		{
			name: "version casing mismatch",
			files: map[string]string{
				"google/cloud/foo/v1p1beta1/service.proto": `syntax = "proto3";
option ruby_package = "Google::Cloud::Foo::V1p1Beta1";
`,
			},
			apiPath: "google/cloud/foo/v1p1beta1",
			wantErr: errRubyPackageVersionMismatch,
		},
		{
			name: "version value mismatch",
			files: map[string]string{
				"google/cloud/foo/v1alpha1/service.proto": `syntax = "proto3";
option ruby_package = "Google::Cloud::Foo::V1";
`,
			},
			apiPath: "google/cloud/foo/v1alpha1",
			wantErr: errRubyPackageVersionMismatch,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			googleapisDir := t.TempDir()
			writeProtoFiles(t, googleapisDir, test.files)

			_, _, err := DefaultGemNames(googleapisDir, test.apiPath)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("got error %v, want errors.Is(_, %v)", err, test.wantErr)
			}
		})
	}
}

var (
	workloadIdentityWrapper = &config.Library{
		Name: "google-cloud-workload_identity",
		APIs: []*config.API{{Path: "google/cloud/workloadidentity/v1"}},
		Ruby: &config.RubyPackage{WrapperOf: []string{"v1:0.0"}},
	}
	workloadIdentityV1 = &config.Library{
		Name: "google-cloud-workload_identity-v1",
		APIs: []*config.API{{Path: "google/cloud/workloadidentity/v1"}},
	}
	backupDRV1 = &config.Library{
		Name: "google-cloud-backupdr-v1",
		APIs: []*config.API{{Path: "google/cloud/backupdr/v1"}},
	}
	workflowsMultiWrapper = &config.Library{
		Name: "google-cloud-workflows",
		APIs: []*config.API{
			{
				Path: "google/cloud/workflows/v1",
				Ruby: &config.RubyAPI{
					RubyCloudOpts: &config.RubyCloudOpts{WrapperOf: "v1:2.0"},
				},
			},
			{
				Path: "google/cloud/workflows/executions/v1",
				Ruby: &config.RubyAPI{
					RubyCloudOpts: &config.RubyCloudOpts{
						GemName:   "google-cloud-workflows-executions",
						WrapperOf: "v1:1.2",
					},
				},
			},
		},
	}
	dialogflowWrapper = &config.Library{
		Name: "google-cloud-dialogflow",
		APIs: []*config.API{{Path: "google/cloud/dialogflow/v2"}},
		Ruby: &config.RubyPackage{WrapperOf: []string{"v2:0.0"}},
	}
)

func TestFindExistingServiceLibrary(t *testing.T) {
	for _, test := range []struct {
		name      string
		libraries []*config.Library
		apiPath   string
		want      *config.Library
	}{
		{
			name:      "versioned library preferred over wrapper",
			libraries: []*config.Library{workloadIdentityWrapper, workloadIdentityV1},
			apiPath:   "google/cloud/workloadidentity/v2",
			want:      workloadIdentityV1,
		},
		{
			name:      "multi-wrapper library for a sub-service",
			libraries: []*config.Library{workflowsMultiWrapper},
			apiPath:   "google/cloud/workflows/executions/v2",
			want:      workflowsMultiWrapper,
		},
		{
			name:      "nested child service of an existing service is net-new",
			libraries: []*config.Library{dialogflowWrapper},
			apiPath:   "google/cloud/dialogflow/cx/v3",
			want:      nil,
		},
		{
			name:    "no libraries",
			apiPath: "google/cloud/workloadidentity/v1",
			want:    nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := FindExistingServiceLibrary(test.libraries, test.apiPath)
			if got != test.want {
				t.Errorf("got %v, want %v", got, test.want)
			}
		})
	}
}

func TestVersionedGemName(t *testing.T) {
	for _, test := range []struct {
		name       string
		serviceLib *config.Library
		apiPath    string
		want       string
	}{
		{
			name:       "versioned library",
			serviceLib: workloadIdentityV1,
			apiPath:    "google/cloud/workloadidentity/v2",
			want:       "google-cloud-workload_identity-v2",
		},
		{
			name:       "custom base gem name is reused",
			serviceLib: backupDRV1,
			apiPath:    "google/cloud/backupdr/v1beta",
			want:       "google-cloud-backupdr-v1beta",
		},
		{
			name:       "wrapper library",
			serviceLib: workloadIdentityWrapper,
			apiPath:    "google/cloud/workloadidentity/v2",
			want:       "google-cloud-workload_identity-v2",
		},
		{
			name:       "multi-wrapper sub-service uses its gem name",
			serviceLib: workflowsMultiWrapper,
			apiPath:    "google/cloud/workflows/executions/v2",
			want:       "google-cloud-workflows-executions-v2",
		},
		{
			name:       "library for a different service",
			serviceLib: backupDRV1,
			apiPath:    "google/cloud/workloadidentity/v2",
			want:       "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := VersionedGemName(test.serviceLib, test.apiPath)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func writeProtoFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for relPath, content := range files {
		fullPath := filepath.Join(root, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
