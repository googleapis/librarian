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
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestModelAnnotations(t *testing.T) {
	model := api.NewTestAPI(
		[]*api.Message{}, []*api.Enum{},
		[]*api.Service{{Name: "Workflows", Package: "google.cloud.workflows.v1"}})
	codec := newTestCodec(t, model, map[string]string{"copyright-year": "2038"})
	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}
	want := &modelAnnotations{
		PackageName:   "GoogleCloudWorkflowsV1",
		CopyrightYear: "2038",
		MonorepoRoot:  ".",
	}
	if diff := cmp.Diff(want, model.Codec, cmpopts.IgnoreFields(modelAnnotations{}, "BoilerPlate", "DependsOn")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestModelAnnotations_MessagesWithWkt(t *testing.T) {
	enum := &api.Enum{
		Name: "SomeEnum", ID: ".test.SomeSnum", Package: "test",
		Values: []*api.EnumValue{{Name: "UNSPECIFIED", Number: 0}},
	}
	enum.UniqueNumberValues = enum.Values
	for _, test := range []struct {
		name  string
		model *api.API
		want  map[string]bool
	}{
		{
			name: "Messages with wkt",
			model: api.NewTestAPI(
				[]*api.Message{{Name: "Request", ID: ".test.Request", Package: "test"}}, nil, nil),
			want: map[string]bool{"GoogleCloudWkt": true},
		},
		{
			name:  "Enum with wkt",
			model: api.NewTestAPI(nil, []*api.Enum{enum}, nil),
			want:  map[string]bool{"GoogleCloudWkt": false},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			codec := newTestCodec(t, test.model, map[string]string{})
			wkt := &Dependency{
				SwiftDependency: config.SwiftDependency{
					ApiPackage: "google.protobuf",
					Name:       "GoogleCloudWkt",
				},
			}
			codec.ApiPackages = map[string]*Dependency{wkt.ApiPackage: wkt}
			codec.Dependencies = []*Dependency{wkt}
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}
			got := map[string]bool{}
			for _, d := range codec.Dependencies {
				got[d.Name] = d.Required
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestModelAnnotations_WithExternalDependencies(t *testing.T) {
	externalMessage := &api.Message{
		Name:    "ExternalMessage",
		Package: "google.cloud.external.v1",
		ID:      ".google.cloud.external.v1.ExternalMessage",
	}

	message := &api.Message{
		Name:    "LocalMessage",
		Package: "google.cloud.test.v1",
		ID:      ".google.cloud.test.v1.LocalMessage",
		Fields: []*api.Field{
			{
				Name:    "ext_field",
				Typez:   api.MESSAGE_TYPE,
				TypezID: ".google.cloud.external.v1.ExternalMessage",
			},
		},
	}

	model := api.NewTestAPI(
		[]*api.Message{message}, []*api.Enum{}, []*api.Service{})
	model.State.MessageByID[externalMessage.ID] = externalMessage
	codec := newTestCodec(t, model, nil)
	dep1 := &Dependency{
		SwiftDependency: config.SwiftDependency{
			ApiPackage: "google.cloud.external.v1",
			Name:       "external-package",
		},
	}
	dep2 := &Dependency{
		SwiftDependency: config.SwiftDependency{
			ApiPackage: "google.cloud.unused.v1",
			Name:       "unused-package",
		},
	}
	codec.ApiPackages = map[string]*Dependency{
		"google.cloud.external.v1": dep1,
		"google.cloud.unused.v1":   dep2,
	}
	codec.Dependencies = []*Dependency{dep1, dep2}

	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	ann, ok := model.Codec.(*modelAnnotations)
	if !ok {
		t.Fatalf("expected model.Codec to be *modelAnnotations, got %T", model.Codec)
	}

	wantDependsOn := map[string]*Dependency{
		"external-package": {
			SwiftDependency: config.SwiftDependency{
				ApiPackage: "google.cloud.external.v1",
				Name:       "external-package",
			},
			Required: true,
		},
	}

	if diff := cmp.Diff(wantDependsOn, ann.DependsOn); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
