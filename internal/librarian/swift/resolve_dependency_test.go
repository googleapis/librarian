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

func TestResolveDependencyVersions_PreservesExplicitVersion(t *testing.T) {
	cfg := &config.Config{
		Libraries: []*config.Library{
			{
				Name:    "google-rpc",
				Version: "0.1.0-preview",
			},
		},
	}
	library := &config.Library{
		Name: "test-lib",
		Swift: &config.SwiftPackage{
			SwiftDefault: config.SwiftDefault{
				Dependencies: []config.SwiftDependency{
					{
						Name:    "GoogleRpc",
						URL:     "https://github.com/googleapis/swift-google-rpc",
						Version: "1.0.0-explicit",
					},
				},
			},
		},
	}

	ResolveDependencyVersions(cfg, library)

	got := library.Swift.Dependencies[0].Version
	want := "1.0.0-explicit"
	if got != want {
		t.Errorf("got version %q, want %q", got, want)
	}
}

func TestResolveDependencyVersions_MatchesByNameOverride(t *testing.T) {
	cfg := &config.Config{
		Libraries: []*config.Library{
			{
				Name:    "google-iam-v1",
				Version: "0.1.0-preview",
				Swift: &config.SwiftPackage{
					LibraryNameOverride: "GoogleIAMV1",
				},
			},
		},
	}
	library := &config.Library{
		Name: "test-lib",
		Swift: &config.SwiftPackage{
			SwiftDefault: config.SwiftDefault{
				Dependencies: []config.SwiftDependency{
					{
						Name: "GoogleIAMV1",
						URL:  "https://github.com/googleapis/swift-google-iam-v1",
					},
				},
			},
		},
	}

	ResolveDependencyVersions(cfg, library)

	got := library.Swift.Dependencies[0].Version
	want := "0.1.0-preview"
	if got != want {
		t.Errorf("got version %q, want %q", got, want)
	}
}

func TestResolveDependencyVersions_MatchesByNameCamelCase(t *testing.T) {
	cfg := &config.Config{
		Libraries: []*config.Library{
			{
				Name:    "google-rpc",
				Version: "0.1.0-preview",
			},
		},
	}
	library := &config.Library{
		Name: "test-lib",
		Swift: &config.SwiftPackage{
			SwiftDefault: config.SwiftDefault{
				Dependencies: []config.SwiftDependency{
					{
						Name: "GoogleRpc",
						URL:  "https://github.com/googleapis/swift-google-rpc",
					},
				},
			},
		},
	}

	ResolveDependencyVersions(cfg, library)

	got := library.Swift.Dependencies[0].Version
	want := "0.1.0-preview"
	if got != want {
		t.Errorf("got version %q, want %q", got, want)
	}
}

func TestResolveDependencyVersions_MatchesByExactName(t *testing.T) {
	cfg := &config.Config{
		Libraries: []*config.Library{
			{
				Name:    "google-rpc",
				Version: "0.1.0-preview",
			},
		},
	}
	library := &config.Library{
		Name: "test-lib",
		Swift: &config.SwiftPackage{
			SwiftDefault: config.SwiftDefault{
				Dependencies: []config.SwiftDependency{
					{
						Name: "google-rpc",
						URL:  "https://github.com/googleapis/swift-google-rpc",
					},
				},
			},
		},
	}

	ResolveDependencyVersions(cfg, library)

	got := library.Swift.Dependencies[0].Version
	want := "0.1.0-preview"
	if got != want {
		t.Errorf("got version %q, want %q", got, want)
	}
}

func TestResolveDependencyVersions_MatchesByCaseInsensitiveCamelCase(t *testing.T) {
	cfg := &config.Config{
		Libraries: []*config.Library{
			{
				Name:    "google-longrunning",
				Version: "0.1.0-preview",
			},
		},
	}
	library := &config.Library{
		Name: "test-lib",
		Swift: &config.SwiftPackage{
			SwiftDefault: config.SwiftDefault{
				Dependencies: []config.SwiftDependency{
					{
						Name: "GoogleLongRunning",
						URL:  "https://github.com/googleapis/swift-google-longrunning",
					},
				},
			},
		},
	}

	ResolveDependencyVersions(cfg, library)

	got := library.Swift.Dependencies[0].Version
	want := "0.1.0-preview"
	if got != want {
		t.Errorf("got version %q, want %q", got, want)
	}
}

func TestResolveDependencyVersions_MatchesByURLRepoName(t *testing.T) {
	cfg := &config.Config{
		Libraries: []*config.Library{
			{
				Name:    "google-cloud-auth",
				Output:  "packages/swift-google-auth/Sources/GoogleCloudAuth/generated",
				Version: "0.0.0-preview",
			},
		},
	}
	library := &config.Library{
		Name: "test-lib",
		Swift: &config.SwiftPackage{
			SwiftDefault: config.SwiftDefault{
				Dependencies: []config.SwiftDependency{
					{
						Name: "CustomAuthName",
						URL:  "https://github.com/googleapis/swift-google-auth.git",
					},
				},
			},
		},
	}

	ResolveDependencyVersions(cfg, library)

	got := library.Swift.Dependencies[0].Version
	want := "0.0.0-preview"
	if got != want {
		t.Errorf("got version %q, want %q", got, want)
	}
}

func TestResolveDependencyVersions_MatchesByPath(t *testing.T) {
	cfg := &config.Config{
		Libraries: []*config.Library{
			{
				Name:    "google-cloud-kms-v1",
				Output:  "generated/swift-google-cloud-kms-v1",
				Version: "0.1.0-preview",
			},
		},
	}
	library := &config.Library{
		Name: "test-lib",
		Swift: &config.SwiftPackage{
			SwiftDefault: config.SwiftDefault{
				Dependencies: []config.SwiftDependency{
					{
						Name: "KMSClient",
						Path: "generated/swift-google-cloud-kms-v1",
					},
				},
			},
		},
	}

	ResolveDependencyVersions(cfg, library)

	got := library.Swift.Dependencies[0].Version
	want := "0.1.0-preview"
	if got != want {
		t.Errorf("got version %q, want %q", got, want)
	}
}

func TestResolveDependencyVersions_MatchesByApiPackage(t *testing.T) {
	cfg := &config.Config{
		Libraries: []*config.Library{
			{
				Name: "custom-rpc",
				APIs: []*config.API{
					{Path: "google/rpc"},
				},
				Version: "0.1.0-preview",
			},
		},
	}
	library := &config.Library{
		Name: "test-lib",
		Swift: &config.SwiftPackage{
			SwiftDefault: config.SwiftDefault{
				Dependencies: []config.SwiftDependency{
					{
						Name:       "RPCClient",
						ApiPackage: "google.rpc",
					},
				},
			},
		},
	}

	ResolveDependencyVersions(cfg, library)

	got := library.Swift.Dependencies[0].Version
	want := "0.1.0-preview"
	if got != want {
		t.Errorf("got version %q, want %q", got, want)
	}
}

func TestResolveDependencyVersions_FallbackToDefaultVersion(t *testing.T) {
	cfg := &config.Config{
		Default: &config.Default{
			Swift: &config.SwiftDefault{
				DefaultVersion: "0.0.0-preview",
			},
		},
		Libraries: []*config.Library{
			{
				Name: "google-rpc",
			},
		},
	}
	library := &config.Library{
		Name: "test-lib",
		Swift: &config.SwiftPackage{
			SwiftDefault: config.SwiftDefault{
				Dependencies: []config.SwiftDependency{
					{
						Name: "GoogleRpc",
						URL:  "https://github.com/googleapis/swift-google-rpc",
					},
				},
			},
		},
	}

	ResolveDependencyVersions(cfg, library)

	got := library.Swift.Dependencies[0].Version
	want := "0.0.0-preview"
	if got != want {
		t.Errorf("got version %q, want %q", got, want)
	}
}

func TestResolveDependencyVersions_ExternalDependencyNoMatch(t *testing.T) {
	cfg := &config.Config{
		Libraries: []*config.Library{
			{
				Name:    "google-rpc",
				Version: "0.1.0-preview",
			},
		},
	}
	library := &config.Library{
		Name: "test-lib",
		Swift: &config.SwiftPackage{
			SwiftDefault: config.SwiftDefault{
				Dependencies: []config.SwiftDependency{
					{
						Name: "SwiftProtobuf",
						URL:  "https://github.com/apple/swift-protobuf",
					},
				},
			},
		},
	}

	ResolveDependencyVersions(cfg, library)

	got := library.Swift.Dependencies[0].Version
	want := ""
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
