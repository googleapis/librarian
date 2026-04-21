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
		WktPackage:    "GoogleCloudWkt",
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
				Typez:   api.TypezMessage,
				TypezID: ".google.cloud.external.v1.ExternalMessage",
			},
		},
	}

	service := &api.Service{
		Name:    "TestService",
		ID:      ".google.cloud.test.v1.TestService",
		Package: "google.cloud.test.v1",
	}

	model := api.NewTestAPI(
		[]*api.Message{message}, []*api.Enum{}, []*api.Service{service})
	model.State.MessageByID[externalMessage.ID] = externalMessage
	codec := newTestCodec(t, model, nil)
	codec.withExtraDependencies(t, []config.SwiftDependency{
		{ApiPackage: "google.cloud.external.v1", Name: "GoogleCloudExternalWithOverrideV1"},
		{ApiPackage: "google.cloud.unused.v1", Name: "GoogleUnusedPackage"},
		{Name: "GoogleCloudGax", RequiredByServices: true},
	})

	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	ann, ok := model.Codec.(*modelAnnotations)
	if !ok {
		t.Fatalf("expected model.Codec to be *modelAnnotations, got %T", model.Codec)
	}

	want := map[string]bool{
		"GoogleCloudExternalWithOverrideV1": true,
		"GoogleCloudGax":                    false, // required by the service, but not messages
		"GoogleCloudWkt":                    true,
	}
	got := map[string]bool{}
	for name, dep := range ann.DependsOn {
		got[name] = dep.Required
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantMessageImports := []string{"GoogleCloudExternalWithOverrideV1", "GoogleCloudWkt"}
	if diff := cmp.Diff(wantMessageImports, ann.MessageImports); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	msg := model.Messages[0]
	msgAnn, ok := msg.Codec.(*messageAnnotations)
	if !ok {
		t.Fatalf("expected message.Codec to be *messageAnnotations, got %T", msg.Codec)
	}
	if msgAnn.Model != ann {
		t.Errorf("expected msgAnn.Model to be %p, got %p", ann, msgAnn.Model)
	}
}

func TestModelAnnotations_IgnoreSelfDependency(t *testing.T) {
	model := api.NewTestAPI(
		[]*api.Message{}, []*api.Enum{}, []*api.Service{{Name: "DummyService", Package: "google.cloud.placeholder.v1"}})
	model.PackageName = "google.cloud.placeholder.v1"
	codec := newTestCodec(t, model, nil)
	codec.withExtraDependencies(t, []config.SwiftDependency{
		{ApiPackage: "google.cloud.placeholder.v1", Name: "GoogleCloudPlaceholderV1"},
		{ApiPackage: "google.cloud.other.v1", Name: "GoogleCloudOtherV1", RequiredByServices: true},
	})
	// Make it required to verify the rest of the code works.
	codec.Dependencies[0].Required = true

	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	ann, ok := model.Codec.(*modelAnnotations)
	if !ok {
		t.Fatalf("expected model.Codec to be *modelAnnotations, got %T", model.Codec)
	}

	// Self dependency should be ignored, other should be present.
	want := map[string]bool{
		"GoogleCloudOtherV1": false, // required by the service, but not messages
	}
	got := map[string]bool{}
	for name, dep := range ann.DependsOn {
		got[name] = dep.Required
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
