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

package java

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestDeriveDistributionName(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		want    string
	}{
		{
			name:    "default case",
			library: &config.Library{Name: "secretmanager"},
			want:    "com.google.cloud:google-cloud-secretmanager",
		},
		{
			name: "groupID override",
			library: &config.Library{
				Name: "secretmanager",
				Java: &config.JavaModule{GroupID: "com.custom"},
			},
			want: "com.custom:google-cloud-secretmanager",
		},
		{
			name: "distributionName override",
			library: &config.Library{
				Name: "secretmanager",
				Java: &config.JavaModule{DistributionNameOverride: "com.google.cloud:google-cloud-secretmanager-v1"},
			},
			want: "com.google.cloud:google-cloud-secretmanager-v1",
		},
		{
			name:    "library name already has prefix",
			library: &config.Library{Name: "google-cloud-secretmanager"},
			want:    "com.google.cloud:google-cloud-secretmanager",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := DeriveDistributionName(test.library)
			if got != test.want {
				t.Errorf("deriveDistributionName() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestProtoGroupID(t *testing.T) {
	for _, test := range []struct {
		name                string
		mainArtifactGroupID string
		want                string
	}{
		{
			name:                "cloud group id",
			mainArtifactGroupID: "com.google.cloud",
			want:                "com.google.api.grpc",
		},
		{
			name:                "analytics group id",
			mainArtifactGroupID: "com.google.analytics",
			want:                "com.google.api.grpc",
		},
		{
			name:                "area120 group id",
			mainArtifactGroupID: "com.google.area120",
			want:                "com.google.api.grpc",
		},
		{
			name:                "non-cloud group id",
			mainArtifactGroupID: "com.google.maps",
			want:                "com.google.maps.api.grpc",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := protoGroupID(test.mainArtifactGroupID)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDeriveLibCoords(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		want    libraryCoordinate
	}{
		{
			name: "default case",
			library: &config.Library{
				Name:    "secretmanager",
				Version: "1.2.3",
			},
			want: libraryCoordinate{
				Gapic: coordinate{
					GroupID:    "com.google.cloud",
					ArtifactID: "google-cloud-secretmanager",
					Version:    "1.2.3",
				},
				Parent: coordinate{
					GroupID:    "com.google.cloud",
					ArtifactID: "google-cloud-secretmanager-parent",
					Version:    "1.2.3",
				},
				Bom: coordinate{
					GroupID:    "com.google.cloud",
					ArtifactID: "google-cloud-secretmanager-bom",
					Version:    "1.2.3",
				},
			},
		},
		{
			name: "with distribution name override",
			library: &config.Library{
				Name:    "secretmanager",
				Version: "1.2.3",
				Java: &config.JavaModule{
					DistributionNameOverride: "com.google.cloud:google-secretmanager",
				},
			},
			want: libraryCoordinate{
				Gapic: coordinate{
					GroupID:    "com.google.cloud",
					ArtifactID: "google-secretmanager",
					Version:    "1.2.3",
				},
				Parent: coordinate{
					GroupID:    "com.google.cloud",
					ArtifactID: "google-secretmanager-parent",
					Version:    "1.2.3",
				},
				Bom: coordinate{
					GroupID:    "com.google.cloud",
					ArtifactID: "google-secretmanager-bom",
					Version:    "1.2.3",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := DeriveLibraryCoordinates(test.library)
			if diff := cmp.Diff(test.want, got, cmp.AllowUnexported(libraryCoordinate{}, coordinate{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDeriveAPICoords(t *testing.T) {
	for _, test := range []struct {
		name      string
		lc        libraryCoordinate
		version   string
		wantProto coordinate
		wantGrpc  coordinate
	}{
		{
			name: "standard cloud mapping",
			lc: libraryCoordinate{
				Gapic: coordinate{
					GroupID:    "com.google.cloud",
					ArtifactID: "google-cloud-secretmanager",
					Version:    "1.2.3",
				},
			},
			version: "v1",
			wantProto: coordinate{
				GroupID:    "com.google.api.grpc",
				ArtifactID: "proto-google-cloud-secretmanager-v1",
				Version:    "1.2.3",
			},
			wantGrpc: coordinate{
				GroupID:    "com.google.api.grpc",
				ArtifactID: "grpc-google-cloud-secretmanager-v1",
				Version:    "1.2.3",
			},
		},
		{
			name: "non-cloud mapping",
			lc: libraryCoordinate{
				Gapic: coordinate{
					GroupID:    "com.google.maps",
					ArtifactID: "google-maps-places",
					Version:    "1.2.3",
				},
			},
			version: "v1",
			wantProto: coordinate{
				GroupID:    "com.google.maps.api.grpc",
				ArtifactID: "proto-google-maps-places-v1",
				Version:    "1.2.3",
			},
			wantGrpc: coordinate{
				GroupID:    "com.google.maps.api.grpc",
				ArtifactID: "grpc-google-maps-places-v1",
				Version:    "1.2.3",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := DeriveAPICoordinates(test.lc, test.version)
			if diff := cmp.Diff(test.wantProto, got.Proto, cmp.AllowUnexported(coordinate{})); diff != "" {
				t.Errorf("proto mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantGrpc, got.GRPC, cmp.AllowUnexported(coordinate{})); diff != "" {
				t.Errorf("grpc mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEnsureCloudPrefix(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{"no prefix", "secretmanager", "google-cloud-secretmanager"},
		{"with prefix", "google-cloud-secretmanager", "google-cloud-secretmanager"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := ensureCloudPrefix(test.input)
			if got != test.want {
				t.Errorf("ensureCloudPrefix(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}
