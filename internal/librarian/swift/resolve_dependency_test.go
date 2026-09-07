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

package swift

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestResolveDependencyVersions_NilGuards(t *testing.T) {
	// None of these should panic.
	ResolveDependencyVersions(nil, nil)
	ResolveDependencyVersions(&config.Config{}, nil)
	ResolveDependencyVersions(&config.Config{}, &config.Library{})
	ResolveDependencyVersions(&config.Config{}, &config.Library{Swift: &config.SwiftPackage{}})
}

func TestResolveDependencyVersions(t *testing.T) {
	for _, test := range []struct {
		name         string
		libraries    []*config.Library
		dependencies []config.SwiftDependency
		defaultCfg   *config.Default
		wantVersion  string
	}{
		{
			name: "PreservesExplicitVersion",
			libraries: []*config.Library{
				{
					Name:    "google-rpc",
					Version: "0.1.0-preview",
				},
			},
			dependencies: []config.SwiftDependency{
				{
					Name:    "GoogleRpc",
					URL:     "https://github.com/googleapis/swift-google-rpc",
					Version: "1.0.0-explicit",
				},
			},
			wantVersion: "1.0.0-explicit",
		},
		{
			name: "MatchesByNameOverride",
			libraries: []*config.Library{
				{
					Name:    "google-iam-v1",
					Version: "0.1.0-preview",
					Swift: &config.SwiftPackage{
						LibraryNameOverride: "GoogleIAMV1",
					},
				},
			},
			dependencies: []config.SwiftDependency{
				{
					Name: "GoogleIAMV1",
					URL:  "https://github.com/googleapis/swift-google-iam-v1",
				},
			},
			wantVersion: "0.1.0-preview",
		},
		{
			name: "MatchesByNameCamelCase",
			libraries: []*config.Library{
				{
					Name:    "google-rpc",
					Version: "0.1.0-preview",
				},
			},
			dependencies: []config.SwiftDependency{
				{
					Name: "GoogleRpc",
					URL:  "https://github.com/googleapis/swift-google-rpc",
				},
			},
			wantVersion: "0.1.0-preview",
		},
		{
			name: "MatchesByExactName",
			libraries: []*config.Library{
				{
					Name:    "google-rpc",
					Version: "0.1.0-preview",
				},
			},
			dependencies: []config.SwiftDependency{
				{
					Name: "google-rpc",
					URL:  "https://github.com/googleapis/swift-google-rpc",
				},
			},
			wantVersion: "0.1.0-preview",
		},
		{
			name: "MatchesByCaseInsensitiveCamelCase",
			libraries: []*config.Library{
				{
					Name:    "google-longrunning",
					Version: "0.1.0-preview",
				},
			},
			dependencies: []config.SwiftDependency{
				{
					Name: "GoogleLongRunning",
					URL:  "https://github.com/googleapis/swift-google-longrunning",
				},
			},
			wantVersion: "0.1.0-preview",
		},
		{
			name: "MatchesByURLRepoName",
			libraries: []*config.Library{
				{
					Name:    "google-cloud-auth",
					Output:  "packages/swift-google-auth/Sources/GoogleCloudAuth/generated",
					Version: "0.0.0-preview",
				},
			},
			dependencies: []config.SwiftDependency{
				{
					Name: "CustomAuthName",
					URL:  "https://github.com/googleapis/swift-google-auth.git",
				},
			},
			wantVersion: "0.0.0-preview",
		},
		{
			name: "MatchesByPath",
			libraries: []*config.Library{
				{
					Name:    "google-cloud-kms-v1",
					Output:  "generated/swift-google-cloud-kms-v1",
					Version: "0.1.0-preview",
				},
			},
			dependencies: []config.SwiftDependency{
				{
					Name: "KMSClient",
					Path: "generated/swift-google-cloud-kms-v1",
				},
			},
			wantVersion: "0.1.0-preview",
		},
		{
			name: "NoFalsePositivePathPrefix",
			libraries: []*config.Library{
				{
					Name:    "kms-v11-lib",
					Output:  "generated/swift-google-cloud-kms-v11",
					Version: "0.2.0",
				},
				{
					Name:    "kms-v1-lib",
					Output:  "generated/swift-google-cloud-kms-v1",
					Version: "0.1.0-preview",
				},
			},
			dependencies: []config.SwiftDependency{
				{
					Name: "KMSClient",
					Path: "./generated/swift-google-cloud-kms-v1",
				},
			},
			wantVersion: "0.1.0-preview",
		},
		{
			name: "MatchesByApiPackage",
			libraries: []*config.Library{
				{
					Name: "custom-rpc",
					APIs: []*config.API{
						{Path: "google/rpc"},
					},
					Version: "0.1.0-preview",
				},
			},
			dependencies: []config.SwiftDependency{
				{
					Name:       "RPCClient",
					ApiPackage: "google.rpc",
				},
			},
			wantVersion: "0.1.0-preview",
		},
		{
			name: "FallbackToDefaultVersion",
			libraries: []*config.Library{
				{
					Name: "google-rpc",
				},
			},
			defaultCfg: &config.Default{
				Swift: &config.SwiftDefault{
					DefaultVersion: "0.0.0-preview",
				},
			},
			dependencies: []config.SwiftDependency{
				{
					Name: "GoogleRpc",
					URL:  "https://github.com/googleapis/swift-google-rpc",
				},
			},
			wantVersion: "0.0.0-preview",
		},
		{
			name: "ExternalDependencyNoMatch",
			libraries: []*config.Library{
				{
					Name:    "google-rpc",
					Version: "0.1.0-preview",
				},
			},
			dependencies: []config.SwiftDependency{
				{
					Name: "SwiftProtobuf",
					URL:  "https://github.com/apple/swift-protobuf",
				},
			},
			wantVersion: "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{
				Libraries: test.libraries,
				Default:   test.defaultCfg,
			}
			library := &config.Library{
				Name: "test-lib",
				Swift: &config.SwiftPackage{
					SwiftDefault: config.SwiftDefault{
						Dependencies: test.dependencies,
					},
				},
			}
			ResolveDependencyVersions(cfg, library)
			got := library.Swift.Dependencies[0].Version
			if diff := cmp.Diff(test.wantVersion, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
