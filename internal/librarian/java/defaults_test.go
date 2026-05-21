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
					"proto-google-cloud-vision-v1/src/main/java/com/google/cloud/vision/v1/ImageName.java",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := Tidy(test.lib)
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
			name: "empty java config",
			lib:  &config.Library{},
		},
		{
			name: "empty distribution name override",
			lib: &config.Library{
				Java: &config.JavaModule{},
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
			name: "omit common resources conflict",
			lib: &config.Library{
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
