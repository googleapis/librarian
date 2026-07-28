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
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestLibraryName(t *testing.T) {
	for _, test := range []struct {
		name   string
		input  string
		config *config.SwiftPackage
		want   string
	}{
		{
			name:   "cloud storage v2",
			input:  "google.cloud.storage.v2",
			config: nil,
			want:   "GoogleCloudStorageV2",
		},
		{
			name:   "iam v1",
			input:  "google.iam.v1",
			config: nil,
			want:   "GoogleIamV1",
		},
		{
			name:   "cloud location",
			input:  "google.cloud.location",
			config: nil,
			want:   "GoogleCloudLocation",
		},
		{
			name:   "api",
			input:  "google.api",
			config: nil,
			want:   "GoogleApi",
		},
		{
			name:   "grafeas v1",
			input:  "grafeas.v1",
			config: nil,
			want:   "GrafeasV1",
		},
		{
			name:   "grafeas v1",
			input:  "grafeas.v1",
			config: &config.SwiftPackage{LibraryNameOverride: "GoogleGrafeasV1"},
			want:   "GoogleGrafeasV1",
		},
		{
			name:  "corner case",
			input: "google",
			want:  "Google",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			model := api.NewTestAPI(nil, nil, nil)
			model.PackageName = test.input
			got, err := LibraryName(model, test.config)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Errorf("mismatch got = %q, want %q", got, test.want)
			}
		})
	}
}

func TestLibraryNameError(t *testing.T) {
	model := api.NewTestAPI(nil, nil, nil)
	got, err := LibraryName(model, nil)
	if err == nil {
		t.Errorf("Expected an error, got: %s", got)
	}
}

func TestLibraryNameConflicts(t *testing.T) {
	for _, test := range []struct {
		name  string
		model *api.API
	}{
		{
			name:  "C#",
			model: api.NewTestAPI(nil, nil, nil).WithCsharpNamespace("Conflict"),
		},
		{
			name:  "PHP",
			model: api.NewTestAPI(nil, nil, nil).WithPhpNamespace("Conflict"),
		},
		{
			name:  "Ruby",
			model: api.NewTestAPI(nil, nil, nil).WithRubyPackage("Conflict"),
		},
		{
			name: "Multiple",
			model: api.NewTestAPI(nil, nil, nil).
				WithCsharpNamespace("CsharpConflict").
				WithPhpNamespace("PHPConflict").
				WithRubyPackage("RubyConflict"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := LibraryName(test.model, nil)
			if err == nil {
				t.Errorf("expected an error, got=%+v", got)
			}
		})
	}
}

func TestLibraryNameOverrideSilencesConflict(t *testing.T) {
	model := api.NewTestAPI(nil, nil, nil).
		WithCsharpNamespace("CsharpConflict").
		WithPhpNamespace("PHPConflict").
		WithRubyPackage("RubyConflict")
	got, err := LibraryName(model, &config.SwiftPackage{LibraryNameOverride: "Override"})
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff("Override", got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
