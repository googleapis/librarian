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
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/semver"
)

func TestFill(t *testing.T) {
	for _, test := range []struct {
		name string
		lib  *config.Library
		want *config.Library
	}{
		{
			name: "fill output from name",
			lib: &config.Library{
				Name: "secretmanager",
			},
			want: &config.Library{
				Name:   "secretmanager",
				Output: "java-secretmanager",
				Java: &config.JavaModule{
					ArtifactID: "google-cloud-secretmanager",
					GroupID:    "com.google.cloud",
				},
			},
		},
		{
			name: "do not overwrite output",
			lib: &config.Library{
				Name:   "secretmanager",
				Output: "custom-output",
			},
			want: &config.Library{
				Name:   "secretmanager",
				Output: "custom-output",
				Java: &config.JavaModule{
					ArtifactID: "google-cloud-secretmanager",
					GroupID:    "com.google.cloud",
				},
			},
		},
		{
			name: "fill samples default",
			lib: &config.Library{
				Name: "secretmanager",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			want: &config.Library{
				Name:   "secretmanager",
				Output: "java-secretmanager",
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
						Java: &config.JavaAPI{
							Samples:               new(true),
							GenerateGAPIC:         new(true),
							GenerateProto:         new(true),
							GenerateGRPC:          new(true),
							GenerateResourceNames: new(true),
						},
					},
				},
				Java: &config.JavaModule{
					ArtifactID: "google-cloud-secretmanager",
					GroupID:    "com.google.cloud",
				},
			},
		},
		{
			name: "do not overwrite samples override",
			lib: &config.Library{
				Name: "secretmanager",
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
						Java: &config.JavaAPI{
							Samples: new(false),
						},
					},
				},
			},
			want: &config.Library{
				Name:   "secretmanager",
				Output: "java-secretmanager",
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
						Java: &config.JavaAPI{
							Samples:               new(false),
							GenerateGAPIC:         new(true),
							GenerateProto:         new(true),
							GenerateGRPC:          new(true),
							GenerateResourceNames: new(true),
						},
					},
				},
				Java: &config.JavaModule{
					ArtifactID: "google-cloud-secretmanager",
					GroupID:    "com.google.cloud",
				},
			},
		},
		{
			name: "do not overwrite non-default group id",
			lib: &config.Library{
				Name: "secretmanager",
				Java: &config.JavaModule{
					GroupID: "com.google.custom",
				},
			},
			want: &config.Library{
				Name:   "secretmanager",
				Output: "java-secretmanager",
				Java: &config.JavaModule{
					ArtifactID: "google-cloud-secretmanager",
					GroupID:    "com.google.custom",
				},
			},
		},
		{
			name: "fill default artifact id",
			lib: &config.Library{
				Name: "secretmanager",
			},
			want: &config.Library{
				Name:   "secretmanager",
				Output: "java-secretmanager",
				Java: &config.JavaModule{
					ArtifactID: "google-cloud-secretmanager",
					GroupID:    "com.google.cloud",
				},
			},
		},
		{
			name: "do not overwrite artifact id",
			lib: &config.Library{
				Name: "secretmanager",
				Java: &config.JavaModule{
					ArtifactID: "custom-secretmanager",
				},
			},
			want: &config.Library{
				Name:   "secretmanager",
				Output: "java-secretmanager",
				Java: &config.JavaModule{
					ArtifactID: "custom-secretmanager",
					GroupID:    "com.google.cloud",
				},
			},
		},
		{
			name: "fill released version from version",
			lib: &config.Library{
				Name:    "secretmanager",
				Version: "1.2.0-SNAPSHOT",
			},
			want: &config.Library{
				Name:    "secretmanager",
				Version: "1.2.0-SNAPSHOT",
				Output:  "java-secretmanager",
				Java: &config.JavaModule{
					ArtifactID:      "google-cloud-secretmanager",
					GroupID:         "com.google.cloud",
					ReleasedVersion: "1.1.0",
				},
			},
		},
		{
			name: "do not fill released version if skip generate",
			lib: &config.Library{
				Name:         "secretmanager",
				Version:      "1.2.0-SNAPSHOT",
				SkipGenerate: true,
			},
			want: &config.Library{
				Name:         "secretmanager",
				Version:      "1.2.0-SNAPSHOT",
				SkipGenerate: true,
				Output:       "java-secretmanager",
				Java: &config.JavaModule{
					ArtifactID: "google-cloud-secretmanager",
					GroupID:    "com.google.cloud",
				},
			},
		},
		{
			name: "do not overwrite released version",
			lib: &config.Library{
				Name:    "secretmanager",
				Version: "1.2.0-SNAPSHOT",
				Java: &config.JavaModule{
					ReleasedVersion: "1.1.5",
				},
			},
			want: &config.Library{
				Name:    "secretmanager",
				Version: "1.2.0-SNAPSHOT",
				Output:  "java-secretmanager",
				Java: &config.JavaModule{
					ArtifactID:      "google-cloud-secretmanager",
					GroupID:         "com.google.cloud",
					ReleasedVersion: "1.1.5",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := Fill(test.lib)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTidy(t *testing.T) {
	for _, test := range []struct {
		name string
		lib  *config.Library
		want *config.Library
	}{
		{
			name: "tidy default output",
			lib: &config.Library{
				Name:   "secretmanager",
				Output: "java-secretmanager",
			},
			want: &config.Library{
				Name:   "secretmanager",
				Output: "",
			},
		},
		{
			name: "do not tidy custom output",
			lib: &config.Library{
				Name:   "secretmanager",
				Output: "custom-output",
			},
			want: &config.Library{
				Name:   "secretmanager",
				Output: "custom-output",
			},
		},
		{
			name: "tidy flags default",
			lib: &config.Library{
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
						Java: &config.JavaAPI{
							Samples:               new(true),
							GenerateGAPIC:         new(true),
							GenerateProto:         new(true),
							GenerateGRPC:          new(true),
							GenerateResourceNames: new(true),
						},
					},
				},
			},
			want: &config.Library{
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
					},
				},
			},
		},
		{
			name: "do not tidy false flags",
			lib: &config.Library{
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
						Java: &config.JavaAPI{
							Samples:               new(false),
							GenerateGAPIC:         new(false),
							GenerateProto:         new(false),
							GenerateGRPC:          new(false),
							GenerateResourceNames: new(false),
						},
					},
				},
			},
			want: &config.Library{
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
						Java: &config.JavaAPI{
							Samples:               new(false),
							GenerateGAPIC:         new(false),
							GenerateProto:         new(false),
							GenerateGRPC:          new(false),
							GenerateResourceNames: new(false),
						},
					},
				},
			},
		},
		{
			name: "tidy default grpc when proto is false",
			lib: &config.Library{
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
						Java: &config.JavaAPI{
							GenerateProto: new(false),
							GenerateGRPC:  new(true),
						},
					},
				},
			},
			want: &config.Library{
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
						Java: &config.JavaAPI{
							GenerateProto: new(false),
						},
					},
				},
			},
		},
		{
			name: "tidy empty additional protos",
			lib: &config.Library{
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
						Java: &config.JavaAPI{
							AdditionalProtos: []*config.AdditionalProto{
								{Path: ""},
								{Path: "google/cloud/common_resources.proto"},
							},
						},
					},
				},
			},
			want: &config.Library{
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
						Java: &config.JavaAPI{
							AdditionalProtos: []*config.AdditionalProto{
								{Path: "google/cloud/common_resources.proto"},
							},
						},
					},
				},
			},
		},
		{
			name: "tidy nil additional protos",
			lib: &config.Library{
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
						Java: &config.JavaAPI{
							AdditionalProtos: []*config.AdditionalProto{
								nil,
							},
						},
					},
				},
			},
			want: &config.Library{
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
					},
				},
			},
		},
		{
			name: "tidy default group id",
			lib: &config.Library{
				Java: &config.JavaModule{
					GroupID: "com.google.cloud",
				},
			},
			want: &config.Library{},
		},
		{
			name: "do not tidy custom group id",
			lib: &config.Library{
				Java: &config.JavaModule{
					GroupID: "com.google.analytics",
				},
			},
			want: &config.Library{
				Java: &config.JavaModule{
					GroupID: "com.google.analytics",
				},
			},
		},
		{
			name: "tidy redundant keep files",
			lib: &config.Library{
				Name: "vision",
				APIs: []*config.API{
					{
						Path: "google/cloud/vision/v1",
					},
				},
				Java: &config.JavaModule{
					GroupID:    "com.google.cloud",
					ArtifactID: "google-cloud-vision",
				},
				Keep: []string{
					"google-cloud-vision/src/main/java/com/google/cloud/vision/v1/stub/Version.java",
					"google-cloud-vision/src/test/java/com/google/cloud/vision/it/ITSystemTest.java",
					"google-cloud-vision/src/test/java/com/google/cloud/vision/v1/it/ITSystemTest.java",
					"google-cloud-vision/src/test/resources/placeholder.txt",
					"google-cloud-vision/src/main/resources/META-INF/native-image/reflect-config.json",
					"proto-google-cloud-vision-v1/src/main/java/com/google/cloud/vision/v1/ImageName.java",
				},
			},
			want: &config.Library{
				Name: "vision",
				APIs: []*config.API{
					{
						Path: "google/cloud/vision/v1",
					},
				},
				Keep: []string{
					"google-cloud-vision/src/main/resources/META-INF/native-image/reflect-config.json",
					"google-cloud-vision/src/test/java/com/google/cloud/vision/it/ITSystemTest.java",
					"google-cloud-vision/src/test/resources/placeholder.txt",
					"proto-google-cloud-vision-v1/src/main/java/com/google/cloud/vision/v1/ImageName.java",
				},
			},
		},
		{
			name: "tidy released version if same as derived",
			lib: &config.Library{
				Name:    "secretmanager",
				Version: "1.2.0-SNAPSHOT",
				Java: &config.JavaModule{
					ReleasedVersion: "1.1.0",
				},
			},
			want: &config.Library{
				Name:    "secretmanager",
				Version: "1.2.0-SNAPSHOT",
			},
		},
		{
			name: "do not tidy released version if different from derived",
			lib: &config.Library{
				Name:    "secretmanager",
				Version: "1.2.0-SNAPSHOT",
				Java: &config.JavaModule{
					ReleasedVersion: "1.1.2",
				},
			},
			want: &config.Library{
				Name:    "secretmanager",
				Version: "1.2.0-SNAPSHOT",
				Java: &config.JavaModule{
					ReleasedVersion: "1.1.2",
				},
			},
		},
		{
			name: "do not tidy released version if version is not a snapshot",
			lib: &config.Library{
				Name:    "secretmanager",
				Version: "1.2.0",
				Java: &config.JavaModule{
					ReleasedVersion: "1.2.0",
				},
			},
			want: &config.Library{
				Name:    "secretmanager",
				Version: "1.2.0",
				Java: &config.JavaModule{
					ReleasedVersion: "1.2.0",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := Tidy(test.lib)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	for _, test := range []struct {
		name string
		lib  *config.Library
	}{
		{
			name: "valid java config",
			lib: &config.Library{
				Name: "secretmanager",
				Java: &config.JavaModule{
					ReleasedVersion: "1.2.3",
				},
			},
		},
		{
			name: "skipped library does not require released_version",
			lib: &config.Library{
				Name:         "google-cloud-java",
				SkipGenerate: true,
			},
		},
		{
			name: "valid java config with derivable released version",
			lib: &config.Library{
				Name:    "secretmanager",
				Version: "1.2.0-SNAPSHOT",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := Validate(test.lib); err != nil {
				t.Errorf("Validate(%+v) error = %v, want nil", test.lib, err)
			}
		})
	}
}

func TestValidate_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		lib     *config.Library
		wantErr error
	}{
		{
			name: "invalid version",
			lib: &config.Library{
				Name:    "secretmanager",
				Version: "invalid-semver",
			},
			wantErr: semver.ErrInvalidVersion,
		},
		{
			name: "invalid version with skip generate",
			lib: &config.Library{
				Name:         "secretmanager",
				Version:      "invalid-semver",
				SkipGenerate: true,
			},
			wantErr: semver.ErrInvalidVersion,
		},
		{
			name: "invalid released version",
			lib: &config.Library{
				Name: "secretmanager",
				Java: &config.JavaModule{
					ReleasedVersion: "invalid-semver",
				},
			},
			wantErr: semver.ErrInvalidVersion,
		},
		{
			name: "omit common resources conflict",
			lib: &config.Library{
				Name: "secretmanager",
				Java: &config.JavaModule{
					ReleasedVersion: "1.2.3",
				},
				APIs: []*config.API{
					{
						Path: "google/cloud/conflict/v1",
						Java: &config.JavaAPI{
							OmitCommonResources: true,
							AdditionalProtos: []*config.AdditionalProto{
								{Path: "google/cloud/common_resources.proto"},
							},
						},
					},
				},
			},
			wantErr: ErrOmitCommonResourcesConflict,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := Validate(test.lib)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("Validate() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestTidyKeep(t *testing.T) {
	for _, test := range []struct {
		name string
		keep []string
		want []string
	}{
		{
			name: "nil keep",
			keep: nil,
			want: nil,
		},
		{
			name: "empty keep",
			keep: []string{},
			want: nil,
		},
		{
			name: "no redundant files",
			keep: []string{"foo/bar.java", "baz/qux.java"},
			want: []string{"baz/qux.java", "foo/bar.java"},
		},
		{
			name: "redundant files and sorting",
			keep: []string{
				"google-cloud-vision/src/main/java/com/google/cloud/vision/v1/stub/Version.java",
				"google-cloud-vision/src/test/java/com/google/cloud/vision/it/ITSystemTest.java",
				"google-cloud-vision/src/test/java/com/google/cloud/vision/v1/it/ITSystemTest.java",
				"google-cloud-vision/src/test/resources/placeholder.txt",
				"google-cloud-vision/src/main/resources/META-INF/native-image/reflect-config.json",
				"proto-google-cloud-vision-v1/src/main/java/com/google/cloud/vision/v1/ImageName.java",
			},
			want: []string{
				"google-cloud-vision/src/main/resources/META-INF/native-image/reflect-config.json",
				"google-cloud-vision/src/test/java/com/google/cloud/vision/it/ITSystemTest.java",
				"google-cloud-vision/src/test/resources/placeholder.txt",
				"proto-google-cloud-vision-v1/src/main/java/com/google/cloud/vision/v1/ImageName.java",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := tidyKeep(test.keep)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDeriveLastReleasedVersion(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
	}{
		{input: "1.2.0-SNAPSHOT", want: "1.1.0"},
		{input: "1.10.0-SNAPSHOT", want: "1.9.0"},
		{input: "0.87.0-SNAPSHOT", want: "0.86.0"},
		{input: "0.0.1-SNAPSHOT", want: "0.0.0"},
		{input: "1.10.1-SNAPSHOT", want: "1.10.0"},
		{input: "0.214.0-beta-SNAPSHOT", want: "0.213.0-beta"},
		{input: "0.214.0-beta", want: "0.214.0-beta"},
		{input: "1.2.3", want: "1.2.3"},
	} {
		t.Run(test.input, func(t *testing.T) {
			got, err := deriveLastReleasedVersion(test.input)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDeriveLastReleasedVersion_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		input   string
		wantErr error
	}{
		{
			name:    "invalid version",
			input:   "1.invalid.0-SNAPSHOT",
			wantErr: semver.ErrInvalidVersion,
		},
		{
			name:    "v1.0.0 snapshot",
			input:   "1.0.0-SNAPSHOT",
			wantErr: ErrCannotDeriveReleasedVersion,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := deriveLastReleasedVersion(test.input)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
