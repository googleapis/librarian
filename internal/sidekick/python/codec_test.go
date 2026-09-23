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
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestNewCodec(t *testing.T) {
	for _, test := range []struct {
		name    string
		model   *api.API
		library *config.Library
		want    *codec
	}{
		{
			name: "derived from library configuration",
			model: func() *api.API {
				m := api.NewTestAPI(nil, nil, nil)
				m.Name = "google-cloud-secretmanager"
				m.PackageName = "google.cloud.secretmanager.v1"
				return m
			}(),
			library: &config.Library{
				Name:          "google-cloud-secretmanager",
				Version:       "2.1.0",
				CopyrightYear: "2026",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
				Python: &config.PythonPackage{
					DefaultVersion: "v1",
				},
			},
			want: &codec{
				GenerationYear: "2026",
				PackageName:    "google-cloud-secretmanager",
				PackageVersion: "2.1.0",
				DefaultVersion: "v1",
				CurrentVersion: "v1",
				GAPICNamespace: "google.cloud",
				GAPICName:      "secretmanager",
			},
		},
		{
			name: "fallback to model name when library name is empty",
			model: func() *api.API {
				m := api.NewTestAPI(nil, nil, nil)
				m.Name = "google-cloud-secretmanager"
				return m
			}(),
			library: &config.Library{
				CopyrightYear: "2026",
			},
			want: &codec{
				GenerationYear: "2026",
				PackageName:    "google-cloud-secretmanager",
				PackageVersion: "0.0.0",
			},
		},
		{
			name: "fallback to model package name when library is nil",
			model: func() *api.API {
				m := api.NewTestAPI(nil, nil, nil)
				m.Name = ""
				m.PackageName = "google.cloud.speech.v1"
				return m
			}(),
			library: nil,
			want: &codec{
				GenerationYear: fmt.Sprintf("%04d", time.Now().Year()),
				PackageName:    "google.cloud.speech.v1",
				PackageVersion: "0.0.0",
				DefaultVersion: "v1",
				CurrentVersion: "v1",
				GAPICNamespace: "google.cloud",
				GAPICName:      "speech",
			},
		},
		{
			name: "single library API fallback",
			model: func() *api.API {
				m := api.NewTestAPI(nil, nil, nil)
				m.Name = "google-cloud-redis"
				return m
			}(),
			library: &config.Library{
				APIs: []*config.API{
					{Path: "google/cloud/redis/v1"},
				},
			},
			want: &codec{
				GenerationYear: fmt.Sprintf("%04d", time.Now().Year()),
				PackageName:    "google-cloud-redis",
				PackageVersion: "0.0.0",
				DefaultVersion: "v1",
				CurrentVersion: "v1",
				GAPICNamespace: "google.cloud",
				GAPICName:      "redis",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := newCodec(test.model, test.library, "")
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got, cmpopts.IgnoreFields(codec{}, "Model", "Library", "OutDir", "protoOptionsCache")); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewCodec_Error(t *testing.T) {
	_, err := newCodec(nil, nil, "")
	if !errors.Is(err, ErrNilModel) {
		t.Errorf("newCodec(nil, nil, \"\") error = %v, want %v", err, ErrNilModel)
	}
}

func TestCodec_PackageDir(t *testing.T) {
	for _, test := range []struct {
		name string
		c    *codec
		want string
	}{
		{
			name: "with namespace, name and version",
			c: &codec{
				GAPICNamespace: "google.cloud",
				GAPICName:      "redis",
				CurrentVersion: "v1",
			},
			want: "google/cloud/redis_v1",
		},
		{
			name: "fallback to default version",
			c: &codec{
				GAPICNamespace: "google.cloud",
				GAPICName:      "redis",
				DefaultVersion: "v1",
			},
			want: "google/cloud/redis_v1",
		},
		{
			name: "without version",
			c: &codec{
				GAPICNamespace: "google.cloud",
				GAPICName:      "redis",
			},
			want: "google/cloud/redis",
		},
		{
			name: "empty namespace and name",
			c:    &codec{},
			want: "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.c.packageDir()
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCodec_RootPackageDir(t *testing.T) {
	for _, test := range []struct {
		name string
		c    *codec
		want string
	}{
		{
			name: "standard package",
			c: &codec{
				GAPICNamespace: "google.cloud",
				GAPICName:      "redis",
			},
			want: "google/cloud/redis",
		},
		{
			name: "empty namespace and name",
			c:    &codec{},
			want: "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.c.rootPackageDir()
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCodec_IsDefaultVersion(t *testing.T) {
	for _, test := range []struct {
		name string
		c    *codec
		want bool
	}{
		{
			name: "matching versions",
			c:    &codec{CurrentVersion: "v1", DefaultVersion: "v1"},
			want: true,
		},
		{
			name: "empty default version",
			c:    &codec{CurrentVersion: "v1", DefaultVersion: ""},
			want: true,
		},
		{
			name: "empty current version",
			c:    &codec{CurrentVersion: "", DefaultVersion: "v1"},
			want: true,
		},
		{
			name: "differing versions",
			c:    &codec{CurrentVersion: "v1beta1", DefaultVersion: "v1"},
			want: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.c.isDefaultVersion()
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCodec_ResolveTypeModule(t *testing.T) {
	msg := api.NewTestMessage("Secret").
		WithPackage("google.cloud.secretmanager.v1").
		WithSourceLocation("google/cloud/secretmanager/v1/resources.proto", 10)
	enum := api.NewTestEnum("SecretVersionState").
		WithPackage("google.cloud.secretmanager.v1").
		WithSourceLocation("google/cloud/secretmanager/v1/enums.proto", 20)
	svc := api.NewTestService("SecretManagerService")
	model := api.NewTestAPI([]*api.Message{msg}, []*api.Enum{enum}, []*api.Service{svc})

	cWithModel := &codec{Model: model}
	cWithoutModel := &codec{}

	for _, test := range []struct {
		name    string
		c       *codec
		typeID  string
		service *api.Service
		want    string
	}{
		{
			name:    "empty ID with service falls back to service snake_case",
			c:       cWithModel,
			typeID:  "",
			service: svc,
			want:    "secret_manager_service",
		},
		{
			name:    "empty ID without service falls back to common",
			c:       cWithModel,
			typeID:  "",
			service: nil,
			want:    "common",
		},
		{
			name:    "with Model message returns proto file stem",
			c:       cWithModel,
			typeID:  msg.ID,
			service: svc,
			want:    "resources",
		},
		{
			name:    "with Model enum returns proto file stem",
			c:       cWithModel,
			typeID:  enum.ID,
			service: svc,
			want:    "enums",
		},
		{
			name:    "with Model nested type walks up to parent message",
			c:       cWithModel,
			typeID:  msg.ID + ".SubField",
			service: svc,
			want:    "resources",
		},
		{
			name:    "without Model falls back to service snake_case",
			c:       cWithoutModel,
			typeID:  ".test.SomeMessage",
			service: svc,
			want:    "secret_manager_service",
		},
		{
			name:    "without Model and nil service falls back to common",
			c:       cWithoutModel,
			typeID:  ".test.SomeMessage",
			service: nil,
			want:    "common",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.c.resolveTypeModule(test.typeID, test.service)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func newTestCodec(t *testing.T, model *api.API, library *config.Library) *codec {
	t.Helper()
	c, err := newCodec(model, library, "")
	if err != nil {
		t.Fatal(err)
	}
	return c
}
