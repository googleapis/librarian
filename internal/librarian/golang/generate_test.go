// Copyright 2025 Google LLC
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

package golang

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

const googleapisDir = "../../testdata/googleapis"

func TestCreateProtocOptions(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		channel *config.Channel
		library *config.Library
		want    []string
		wantErr bool
	}{
		{
			name:    "basic case",
			channel: &config.Channel{Path: "google/cloud/secretmanager/v1"},
			library: &config.Library{},
			want: []string{
				"--go_out=outdir",
				"--go_opt=paths=source_relative",
				"--go-grpc_out=outdir",
				"--go-grpc_opt=paths=source_relative",
				"--go_gapic_out=outdir",
				"--go_gapic_opt=metadata,grpc-service-config=google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json,api-service-config=google/cloud/secretmanager/v1/secretmanager_v1.yaml",
			},
		},
		{
			name:    "with transport",
			channel: &config.Channel{Path: "google/cloud/secretmanager/v1"},
			library: &config.Library{Transport: "grpc"},
			want: []string{
				"--go_out=outdir",
				"--go_opt=paths=source_relative",
				"--go-grpc_out=outdir",
				"--go-grpc_opt=paths=source_relative",
				"--go_gapic_out=outdir",
				"--go_gapic_opt=transport=grpc,metadata,grpc-service-config=google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json,api-service-config=google/cloud/secretmanager/v1/secretmanager_v1.yaml",
			},
		},
		{
			name:    "with version",
			channel: &config.Channel{Path: "google/cloud/secretmanager/v1"},
			library: &config.Library{Version: "1.2.3"},
			want: []string{
				"--go_out=outdir",
				"--go_opt=paths=source_relative",
				"--go-grpc_out=outdir",
				"--go-grpc_opt=paths=source_relative",
				"--go_gapic_out=outdir",
				"--go_gapic_opt=metadata,gapic-version=1.2.3,grpc-service-config=google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json,api-service-config=google/cloud/secretmanager/v1/secretmanager_v1.yaml",
			},
		},
		{
			name:    "with go api options",
			channel: &config.Channel{Path: "google/cloud/secretmanager/v1"},
			library: &config.Library{
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							Path:         "google/cloud/secretmanager/v1",
							DisableGAPIC: true,
							ProtoPackage: "google.cloud.secretmanager.v1",
						},
					},
				},
			},
			want: []string{
				"--go_out=outdir",
				"--go_opt=paths=source_relative",
				"--go-grpc_out=outdir",
				"--go-grpc_opt=paths=source_relative",
				"--go_gapic_out=outdir",
				"--go_gapic_opt=metadata,grpc-service-config=google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json,api-service-config=google/cloud/secretmanager/v1/secretmanager_v1.yaml,disable-gapic,proto-package=google.cloud.secretmanager.v1",
			},
		},
		{
			name:    "with nested protos",
			channel: &config.Channel{Path: "google/cloud/secretmanager/v1"},
			library: &config.Library{
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							Path:         "google/cloud/secretmanager/v1",
							NestedProtos: []string{"google/cloud/location/locations.proto", "google/iam/v1/iam_policy.proto"},
						},
					},
				},
			},
			want: []string{
				"--go_out=outdir",
				"--go_opt=paths=source_relative",
				"--go-grpc_out=outdir",
				"--go-grpc_opt=paths=source_relative",
				"--go_gapic_out=outdir",
				"--go_gapic_opt=metadata,grpc-service-config=google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json,api-service-config=google/cloud/secretmanager/v1/secretmanager_v1.yaml,nested-protos=google/cloud/location/locations.proto,google/iam/v1/iam_policy.proto",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := createProtocOptions(test.channel, test.library, googleapisDir, "outdir")
			if (err != nil) != test.wantErr {
				t.Errorf("createProtocOptions() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
