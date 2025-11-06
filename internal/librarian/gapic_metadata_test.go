// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package librarian

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
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

	for _, test := range []struct {
		name    string
		files   map[string][]byte
		library *config.LibraryState
		want    map[string]*gapic.GapicMetadata
	}{
		{
			name: "single metadata file",
			files: map[string][]byte{
				"src/v1/gapic_metadata.json": libv1JSON,
			},
			library: &config.LibraryState{
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
			library: &config.LibraryState{
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
			library: &config.LibraryState{
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
			library: &config.LibraryState{
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
					ServiceVersion: map[string]string{
						"Library": "v1",
					},
					VersionServices: map[string][]string{
						"v1": {"Library"},
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
					ServiceVersion: map[string]string{
						"Library":    "v1",
						"Management": "v1",
					},
					VersionServices: map[string][]string{
						"v1": {"Library", "Management"},
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
					ServiceVersion: map[string]string{
						"ServiceA": "v1",
						"ServiceB": "v1beta1",
					},
					VersionServices: map[string][]string{
						"v1":      {"ServiceA"},
						"v1beta1": {"ServiceB"},
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
					ServiceVersion: map[string]string{
						"Library": "v1",
					},
					VersionServices: map[string][]string{
						"v1": {"Library"},
					},
				},
				{
					LibraryPackage: "cloud.google.com/go/library/apiv2",
					ServiceVersion: map[string]string{
						"AnotherLibraryService": "v2",
					},
					VersionServices: map[string][]string{
						"v2": {"AnotherLibraryService"},
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
			want: []*libraryPackageAPIVersions{
				{
					LibraryPackage:  "cloud.google.com/go/library/apiv1",
					ServiceVersion:  map[string]string{},
					VersionServices: map[string][]string{},
				},
			},
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
					ServiceVersion: map[string]string{
						"LibraryService": "2025-09-14",
						"BookService":    "2025-04-04",
					},
					VersionServices: map[string][]string{
						"2025-09-14": {"LibraryService"},
						"2025-04-04": {"BookService"},
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
					ServiceVersion: map[string]string{
						"LibraryService": "2025-09-14",
						"BookService":    "2025-04-04",
					},
					VersionServices: map[string][]string{
						"2025-09-14": {"LibraryService"},
						"2025-04-04": {"BookService"},
					},
				},
				{
					LibraryPackage: "cloud.google.com/go/anotherlibrary/apiv1",
					ServiceVersion: map[string]string{
						"AnotherLibraryService": "2025-05-24",
					},
					VersionServices: map[string][]string{
						"2025-05-24": {"AnotherLibraryService"},
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
					ServiceVersion: map[string]string{
						"LibraryService":    "2025-09-14",
						"BookService":       "2025-09-14",
						"ShelfService":      "2025-09-14",
						"BookMobileService": "2025-09-14",
						"CounterService":    "2025-09-14",
					},
					VersionServices: map[string][]string{
						"2025-09-14": {"BookService", "BookMobileService", "CounterService", "LibraryService", "ShelfService"},
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
			got, _ := formatAPIVersionReleaseNotes(test.in)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
