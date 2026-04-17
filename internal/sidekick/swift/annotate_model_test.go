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

	model.State = &api.APIState{
		MessageByID: map[string]*api.Message{
			".google.cloud.external.v1.ExternalMessage": externalMessage,
		},
	}

	codec := newTestCodec(t, model, nil)
	dep := &Dependency{
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
		"google.cloud.external.v1": dep,
		"google.cloud.unused.v1":   dep2,
	}
	codec.Dependencies = []*Dependency{dep, dep2}

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
