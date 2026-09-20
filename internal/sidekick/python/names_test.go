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

package python

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestSnakeCase(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{name: "camelCase", input: "getSecret", want: "get_secret"},
		{name: "pascalCase", input: "SecretManager", want: "secret_manager"},
		{name: "already_snake", input: "already_snake", want: "already_snake"},
		{name: "already_snake_with_digits", input: "data_crc32c", want: "data_crc32c"},
		{name: "with_acronym", input: "GetIAMPolicy", want: "get_iam_policy"},
		{name: "kebab-case", input: "foo-bar", want: "foo_bar"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := snakeCase(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPascalCase(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{name: "snake_case", input: "secret_manager", want: "SecretManager"},
		{name: "pascal", input: "SecretManager", want: "SecretManager"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := pascalCase(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPythonIdentifier(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{name: "keyword_from", input: "from", want: "from_"},
		{name: "keyword_import", input: "import", want: "import_"},
		{name: "non_keyword", input: "name", want: "name"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := pythonIdentifier(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsVersionSegment(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  bool
	}{
		{name: "v1", input: "v1", want: true},
		{name: "v1alpha1", input: "v1alpha1", want: true},
		{name: "v1beta1", input: "v1beta1", want: true},
		{name: "v2", input: "v2", want: true},
		{name: "v1p1beta1", input: "v1p1beta1", want: true},
		{name: "invalid", input: "invalid", want: false},
		{name: "v", input: "v", want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := isVersionSegment(test.input)
			if got != test.want {
				t.Errorf("isVersionSegment(%q) = %v, want %v", test.input, got, test.want)
			}
		})
	}
}

func TestPackageInitials(t *testing.T) {
	for _, test := range []struct {
		name string
		pkg  string
		want string
	}{
		{name: "secretmanager", pkg: "google.cloud.secretmanager.v1alpha1", want: "gcsm"},
		{name: "asset", pkg: "google.cloud.asset.v1", want: "gca"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := packageInitials(test.pkg)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFileNeedsAlias(t *testing.T) {
	msgWithMatch := api.NewTestMessage("ExportAssetsRequest").
		WithSourceLocation("google/cloud/asset/v1/asset_service.proto", 1).
		WithFields(
			api.NewTestField("assets").WithType(api.TypezString),
		)
	msgWithoutMatch := api.NewTestMessage("AnotherRequest").
		WithSourceLocation("google/cloud/asset/v1/other.proto", 1).
		WithFields(
			api.NewTestField("foo").WithType(api.TypezString),
		)

	model := api.NewTestAPI([]*api.Message{msgWithMatch, msgWithoutMatch}, nil, nil)
	c := &codec{Model: model}

	for _, test := range []struct {
		name       string
		sourceFile string
		targetStem string
		want       bool
	}{
		{
			name:       "matching field name returns true",
			sourceFile: "google/cloud/asset/v1/asset_service.proto",
			targetStem: "assets",
			want:       true,
		},
		{
			name:       "non-matching field name returns false",
			sourceFile: "google/cloud/asset/v1/other.proto",
			targetStem: "assets",
			want:       false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := c.fileNeedsAlias(test.sourceFile, test.targetStem)
			if got != test.want {
				t.Errorf("fileNeedsAlias(%q, %q) = %v, want %v", test.sourceFile, test.targetStem, got, test.want)
			}
		})
	}
}

func TestDeriveGAPICNamespace(t *testing.T) {
	for _, test := range []struct {
		name    string
		apiPath string
		want    string
	}{
		{
			name:    "google/cloud/redis/v1",
			apiPath: "google/cloud/redis/v1",
			want:    "google.cloud",
		},
		{
			name:    "google/iam/credentials/v1",
			apiPath: "google/iam/credentials/v1",
			want:    "google.iam",
		},
		{
			name:    "google/cloud/aiplatform/v1",
			apiPath: "google/cloud/aiplatform/v1",
			want:    "google.cloud",
		},
		{
			name:    "google/cloud/foo/bar/v1",
			apiPath: "google/cloud/foo/bar/v1",
			want:    "google.cloud",
		},
		{
			name:    "single",
			apiPath: "single",
			want:    "single",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := deriveGAPICNamespace(test.apiPath)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDeriveGAPICName(t *testing.T) {
	for _, test := range []struct {
		name    string
		apiPath string
		want    string
	}{
		{
			name:    "google/cloud/redis/v1",
			apiPath: "google/cloud/redis/v1",
			want:    "redis",
		},
		{
			name:    "google/iam/credentials/v1",
			apiPath: "google/iam/credentials/v1",
			want:    "credentials",
		},
		{
			name:    "google/cloud/foo/bar/v1",
			apiPath: "google/cloud/foo/bar/v1",
			want:    "foo_bar",
		},
		{
			name:    "google/cloud/asset/v1",
			apiPath: "google/cloud/asset/v1",
			want:    "asset",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := deriveGAPICName(test.apiPath)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
