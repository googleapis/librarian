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

package legacylibrarian

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
	gapic "google.golang.org/genproto/googleapis/gapic/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestReadGapicMetadata(t *testing.T) {
	libv1Metadata := &gapic.GapicMetadata{
		LibraryPackage: "cloud.google.com/go/library/apiv1",
		Services: map[string]*gapic.GapicMetadata_ServiceForTransport{
			"Library": {
				ApiVersion: "v1",
			},
		},
	}
	libv1JSON, err := protojson.Marshal(libv1Metadata)
	if err != nil {
		t.Fatalf("protojson.Marshal() failed: %v", err)
	}

	libv2Metadata := &gapic.GapicMetadata{
		LibraryPackage: "cloud.google.com/go/library/apiv2",
		Services: map[string]*gapic.GapicMetadata_ServiceForTransport{
			"AnotherLibraryService": {
				ApiVersion: "v2",
			},
		},
	}
	libv2JSON, err := protojson.Marshal(libv2Metadata)
	if err != nil {
		t.Fatalf("protojson.Marshal() failed: %v", err)
	}

	libTestMetadata := &gapic.GapicMetadata{
		LibraryPackage: "cloud.google.com/go/library/apiv2test",
		Services: map[string]*gapic.GapicMetadata_ServiceForTransport{
			"AnotherLibraryService": {
				ApiVersion: "v2test",
			},
		},
	}
	libTestJSON, err := protojson.Marshal(libTestMetadata)
	if err != nil {
		t.Fatalf("protojson.Marshal() failed: %v", err)
	}

	for _, test := range []struct {
		name    string
		files   map[string][]byte
		library *legacyconfig.LibraryState
		want    map[string]*gapic.GapicMetadata
	}{
		{
			name: "single metadata file",
			files: map[string][]byte{
				"src/v1/gapic_metadata.json": libv1JSON,
			},
			library: &legacyconfig.LibraryState{
				SourceRoots: []string{"src"},
			},
			want: map[string]*gapic.GapicMetadata{
				"cloud.google.com/go/library/apiv1": libv1Metadata,
			},
		},
		{
			name: "multiple metadata files",
			files: map[string][]byte{
				"src/v1/gapic_metadata.json": libv1JSON,
				"src/v2/gapic_metadata.json": libv2JSON,
			},
			library: &legacyconfig.LibraryState{
				SourceRoots: []string{"src"},
			},
			want: map[string]*gapic.GapicMetadata{
				"cloud.google.com/go/library/apiv1": libv1Metadata,
				"cloud.google.com/go/library/apiv2": libv2Metadata,
			},
		},
		{
			name: "multiple metadata files, ignore testdata",
			files: map[string][]byte{
				"src/v1/gapic_metadata.json":   libv1JSON,
				"src/v2/gapic_metadata.json":   libv2JSON,
				"tests/v2/gapic_metadata.json": libTestJSON,
			},
			library: &legacyconfig.LibraryState{
				SourceRoots: []string{"src"},
			},
			want: map[string]*gapic.GapicMetadata{
				"cloud.google.com/go/library/apiv1": libv1Metadata,
				"cloud.google.com/go/library/apiv2": libv2Metadata,
			},
		},
		{
			name: "multiple source roots",
			files: map[string][]byte{
				"src1/v1/gapic_metadata.json": libv1JSON,
				"src2/v2/gapic_metadata.json": libv2JSON,
			},
			library: &legacyconfig.LibraryState{
				SourceRoots: []string{"src1", "src2"},
			},
			want: map[string]*gapic.GapicMetadata{
				"cloud.google.com/go/library/apiv1": libv1Metadata,
				"cloud.google.com/go/library/apiv2": libv2Metadata,
			},
		},
		{
			name: "no metadata files",
			files: map[string][]byte{
				"src/v1/README.md": []byte("Hello, World!"),
			},
			library: &legacyconfig.LibraryState{
				SourceRoots: []string{"src"},
			},
			want: map[string]*gapic.GapicMetadata{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			for path, content := range test.files {
				fullPath := filepath.Join(tmpDir, path)
				if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
					t.Fatalf("os.MkdirAll() failed: %v", err)
				}
				if err := os.WriteFile(fullPath, content, 0644); err != nil {
					t.Fatalf("os.WriteFile() failed: %v", err)
				}
			}

			got, err := readGapicMetadata(tmpDir, test.library)
			if err != nil {
				t.Fatalf("readGapicMetadata() failed: %v", err)
			}
			if diff := cmp.Diff(test.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractAPIVersions(t *testing.T) {
	for _, test := range []struct {
		name string
		in   map[string]*gapic.GapicMetadata
		want []*libraryPackageAPIVersions
	}{
		{
			name: "single service, single version",
			in: map[string]*gapic.GapicMetadata{
				"cloud.google.com/go/library/apiv1": {
					LibraryPackage: "cloud.google.com/go/library/apiv1",
					Services: map[string]*gapic.GapicMetadata_ServiceForTransport{
						"Library": {
							ApiVersion: "v1",
						},
					},
				},
			},
			want: []*libraryPackageAPIVersions{
				{
					LibraryPackage: "cloud.google.com/go/library/apiv1",
					ServiceVersions: []serviceVersion{
						{Service: "Library", Version: "v1"},
					},
					Versions: map[string]bool{
						"v1": true,
					},
				},
			},
		},
		{
			name: "multiple services, same version",
			in: map[string]*gapic.GapicMetadata{
				"cloud.google.com/go/library/apiv1": {
					LibraryPackage: "cloud.google.com/go/library/apiv1",
					Services: map[string]*gapic.GapicMetadata_ServiceForTransport{
						"Library": {
							ApiVersion: "v1",
						},
						"Management": {
							ApiVersion: "v1",
						},
					},
				},
			},
			want: []*libraryPackageAPIVersions{
				{
					LibraryPackage: "cloud.google.com/go/library/apiv1",
					ServiceVersions: []serviceVersion{
						{Service: "Library", Version: "v1"},
						{Service: "Management", Version: "v1"},
					},
					Versions: map[string]bool{
						"v1": true,
					},
				},
			},
		},
		{
			name: "multiple services, different versions",
			in: map[string]*gapic.GapicMetadata{
				"cloud.google.com/go/library/apiv1": {
					LibraryPackage: "cloud.google.com/go/library/apiv1",
					Services: map[string]*gapic.GapicMetadata_ServiceForTransport{
						"ServiceA": {
							ApiVersion: "v1",
						},
						"ServiceB": {
							ApiVersion: "v1beta1",
						},
					},
				},
			},
			want: []*libraryPackageAPIVersions{
				{
					LibraryPackage: "cloud.google.com/go/library/apiv1",
					ServiceVersions: []serviceVersion{
						{Service: "ServiceA", Version: "v1"},
						{Service: "ServiceB", Version: "v1beta1"},
					},
					Versions: map[string]bool{
						"v1":      true,
						"v1beta1": true,
					},
				},
			},
		},
		{
			name: "multiple library packages",
			in: map[string]*gapic.GapicMetadata{
				"cloud.google.com/go/library/apiv1": {
					LibraryPackage: "cloud.google.com/go/library/apiv1",
					Services: map[string]*gapic.GapicMetadata_ServiceForTransport{
						"Library": {
							ApiVersion: "v1",
						},
					},
				},
				"cloud.google.com/go/library/apiv2": {
					LibraryPackage: "cloud.google.com/go/library/apiv2",
					Services: map[string]*gapic.GapicMetadata_ServiceForTransport{
						"AnotherLibraryService": {
							ApiVersion: "v2",
						},
					},
				},
			},
			want: []*libraryPackageAPIVersions{
				{
					LibraryPackage: "cloud.google.com/go/library/apiv1",
					ServiceVersions: []serviceVersion{
						{Service: "Library", Version: "v1"},
					},
					Versions: map[string]bool{
						"v1": true,
					},
				},
				{
					LibraryPackage: "cloud.google.com/go/library/apiv2",
					ServiceVersions: []serviceVersion{
						{Service: "AnotherLibraryService", Version: "v2"},
					},
					Versions: map[string]bool{
						"v2": true,
					},
				},
			},
		},
		{
			name: "empty input map",
			in:   map[string]*gapic.GapicMetadata{},
			want: nil,
		},
		{
			name: "nil input map",
			in:   nil,
			want: nil,
		},
		{
			name: "empty services in metadata",
			in: map[string]*gapic.GapicMetadata{
				"cloud.google.com/go/library/apiv1": {
					LibraryPackage: "cloud.google.com/go/library/apiv1",
					Services:       map[string]*gapic.GapicMetadata_ServiceForTransport{},
				},
			},
			want: nil,
		},
		{
			name: "one service, no api_version",
			in: map[string]*gapic.GapicMetadata{
				"cloud.google.com/go/library/apiv1": {
					LibraryPackage: "cloud.google.com/go/library/apiv1",
					Services:       map[string]*gapic.GapicMetadata_ServiceForTransport{"Library": {}},
				},
			},
			want: nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := extractAPIVersions(test.in)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFormatAPIVersionReleaseNotes(t *testing.T) {
	for _, test := range []struct {
		name string
		in   []*libraryPackageAPIVersions
		want string
	}{
		{
			name: "single library, multiple versions",
			in: []*libraryPackageAPIVersions{
				{
					LibraryPackage: "cloud.google.com/go/library/apiv1",
					ServiceVersions: []serviceVersion{
						{Service: "BookService", Version: "2025-04-04"},
						{Service: "LibraryService", Version: "2025-09-14"},
					},
					Versions: map[string]bool{
						"2025-09-14": true,
						"2025-04-04": true,
					},
				},
			},
			want: `### API Versions

<details><summary>cloud.google.com/go/library/apiv1</summary>

* BookService: 2025-04-04
* LibraryService: 2025-09-14

</details>
`,
		},
		{
			name: "multiple libraries, multiple versions",
			in: []*libraryPackageAPIVersions{
				{
					LibraryPackage: "cloud.google.com/go/library/apiv1",
					ServiceVersions: []serviceVersion{
						{Service: "BookService", Version: "2025-04-04"},
						{Service: "LibraryService", Version: "2025-09-14"},
					},
					Versions: map[string]bool{
						"2025-09-14": true,
						"2025-04-04": true,
					},
				},
				{
					LibraryPackage: "cloud.google.com/go/anotherlibrary/apiv1",
					ServiceVersions: []serviceVersion{
						{Service: "AnotherLibraryService", Version: "2025-05-24"},
					},
					Versions: map[string]bool{
						"2025-05-24": true,
					},
				},
			},
			want: `### API Versions

<details><summary>cloud.google.com/go/library/apiv1</summary>

* BookService: 2025-04-04
* LibraryService: 2025-09-14

</details>


<details><summary>cloud.google.com/go/anotherlibrary/apiv1</summary>

* AnotherLibraryService: 2025-05-24

</details>
`,
		},
		{
			name: "single library, many services, same version",
			in: []*libraryPackageAPIVersions{
				{
					LibraryPackage: "cloud.google.com/go/library/apiv1",
					ServiceVersions: []serviceVersion{
						{Service: "BookService", Version: "2025-09-14"},
						{Service: "BookMobileService", Version: "2025-09-14"},
						{Service: "CounterService", Version: "2025-09-14"},
						{Service: "LibraryService", Version: "2025-09-14"},
						{Service: "ShelfService", Version: "2025-09-14"},
					},
					Versions: map[string]bool{
						"2025-09-14": true,
					},
				},
			},
			want: `### API Versions

<details><summary>cloud.google.com/go/library/apiv1</summary>

* All: 2025-09-14

</details>
`,
		},
		{
			name: "empty input",
			in:   []*libraryPackageAPIVersions{},
			want: "",
		},
		{
			name: "nil input",
			in:   nil,
			want: "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := formatAPIVersionReleaseNotes(test.in)
			if err != nil {
				t.Fatalf("formatAPIVersionReleaseNotes() failed: %v", err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
