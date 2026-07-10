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

package dart

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestNewOption(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		want    *Option
	}{
		{
			name: "valid configuration",
			library: &config.Library{
				Name:                "google_cloud_secretmanager_v1",
				Version:             "0.1.0",
				Output:              "packages/",
				SpecificationFormat: config.SpecProtobuf,
				CopyrightYear:       "2026",
				SkipRelease:         true,
			},
			want: &Option{
				Name:                "google_cloud_secretmanager_v1",
				Version:             "0.1.0",
				Output:              "packages/",
				SpecificationFormat: config.SpecProtobuf,
				CopyrightYear:       "2026",
				SkipRelease:         true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := NewOption(test.library)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewOption_Error(t *testing.T) {
	library := &config.Library{
		SpecificationFormat: "openapi",
	}
	_, err := NewOption(library)
	if !errors.Is(err, errInvalidSpecificationFormat) {
		t.Fatalf("NewOption() error = %v, wantErr = %v", err, errInvalidSpecificationFormat)
	}
}

func TestVerify(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
	}{
		{
			name: "empty specification format is valid",
			library: &config.Library{
				SpecificationFormat: "",
			},
		},
		{
			name: "protobuf specification format is valid",
			library: &config.Library{
				SpecificationFormat: config.SpecProtobuf,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := verify(test.library); err != nil {
				t.Fatalf("verify() = %v, want nil", err)
			}
		})
	}
}

func TestVerify_Error(t *testing.T) {
	library := &config.Library{
		SpecificationFormat: "openapi",
	}
	err := verify(library)
	if !errors.Is(err, errInvalidSpecificationFormat) {
		t.Fatalf("verify() error = %v, wantErr = %v", err, errInvalidSpecificationFormat)
	}
}
