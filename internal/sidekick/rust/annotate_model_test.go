// Copyright 2025 Google LLC
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

package rust

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	libconfig "github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestDefaultFeatures(t *testing.T) {
	for _, test := range []struct {
		Options map[string]string
		Want    []string
	}{
		{
			Options: map[string]string{
				"per-service-features": "true",
			},
			Want: []string{"service-0", "service-1"},
		},
		{
			Options: map[string]string{
				"per-service-features": "false",
			},
			Want: nil,
		},
		{
			Options: map[string]string{
				"per-service-features": "true",
				"default-features":     "service-1",
			},
			Want: []string{"service-1"},
		},
		{
			Options: map[string]string{
				"per-service-features": "true",
				"default-features":     "",
			},
			Want: []string{},
		},
	} {
		model := newTestAnnotateModelAPI()
		codec := newTestCodec(t, libconfig.SpecProtobuf, "", test.Options)
		got, err := annotateModel(model, codec)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Options=%v", test.Options)
		if diff := cmp.Diff(test.Want, got.DefaultFeatures); diff != "" {
			t.Errorf("mismatch (-want, +got):\n%s", diff)
		}
	}
}

func TestRustdocWarnings(t *testing.T) {
	for _, test := range []struct {
		Options map[string]string
		Want    []string
	}{
		{
			Options: map[string]string{},
			Want:    nil,
		},
		{
			Options: map[string]string{
				"disabled-rustdoc-warnings": "",
			},
			Want: []string{},
		},
		{
			Options: map[string]string{
				"disabled-rustdoc-warnings": "a,b,c",
			},
			Want: []string{"a", "b", "c"},
		},
	} {
		model := newTestAnnotateModelAPI()
		codec := newTestCodec(t, libconfig.SpecProtobuf, "", test.Options)
		got, err := annotateModel(model, codec)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Options=%v", test.Options)
		if diff := cmp.Diff(test.Want, got.DisabledRustdocWarnings); diff != "" {
			t.Errorf("mismatch (-want, +got):\n%s", diff)
		}
	}
}

func TestClippyWarnings(t *testing.T) {
	for _, test := range []struct {
		Options map[string]string
		Want    []string
	}{
		{
			Options: map[string]string{},
			Want:    nil,
		},
		{
			Options: map[string]string{
				"disabled-clippy-warnings": "",
			},
			Want: []string{},
		},
		{
			Options: map[string]string{
				"disabled-clippy-warnings": "a,b,c",
			},
			Want: []string{"a", "b", "c"},
		},
	} {
		model := newTestAnnotateModelAPI()
		codec := newTestCodec(t, libconfig.SpecProtobuf, "", test.Options)
		got, err := annotateModel(model, codec)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Options=%v", test.Options)
		if diff := cmp.Diff(test.Want, got.DisabledClippyWarnings); diff != "" {
			t.Errorf("mismatch (-want, +got):\n%s", diff)
		}
	}
}

func TestInternalBuildersAnnotation(t *testing.T) {
	for _, test := range []struct {
		Options        map[string]string
		Want           bool
		WantVisibility string
	}{
		{
			Options:        map[string]string{},
			Want:           false,
			WantVisibility: "pub",
		},
		{
			Options: map[string]string{
				"internal-builders": "true",
			},
			Want:           true,
			WantVisibility: "pub(crate)",
		},
		{
			Options: map[string]string{
				"internal-builders": "false",
			},
			Want:           false,
			WantVisibility: "pub",
		},
	} {
		model := newTestAnnotateModelAPI()
		codec := newTestCodec(t, libconfig.SpecProtobuf, "", test.Options)
		got, err := annotateModel(model, codec)
		if err != nil {
			t.Fatal(err)
		}
		if got.InternalBuilders != test.Want {
			t.Errorf("mismatch in InternalBuilders, want=%v, got=%v", test.Want, got.InternalBuilders)
		}
		svcAnn := model.Services[0].Codec.(*serviceAnnotations)
		if svcAnn.InternalBuilders != test.Want {
			t.Errorf("mismatch in service InternalBuilders, want=%v, got=%v", test.Want, svcAnn.InternalBuilders)
		}
		if got.BuilderVisibility() != test.WantVisibility {
			t.Errorf("mismatch in BuilderVisibility, want=%s, got=%s", test.WantVisibility, got.BuilderVisibility())
		}
		if svcAnn.BuilderVisibility() != test.WantVisibility {
			t.Errorf("mismatch in service BuilderVisibility, want=%s, got=%s", test.WantVisibility, svcAnn.BuilderVisibility())
		}
	}
}

func newTestAnnotateModelAPI() *api.API {
	service0 := &api.Service{
		Name: "Service0",
		ID:   "..Service0",
		Methods: []*api.Method{
			{
				Name:         "get",
				ID:           "..Service0.get",
				InputTypeID:  ".google.protobuf.Empty",
				OutputTypeID: ".google.protobuf.Empty",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							Verb:         "GET",
							PathTemplate: api.NewPathTemplate().WithLiteral("resource"),
						},
					},
				},
			},
		},
	}
	service1 := &api.Service{
		Name: "Service1",
		ID:   "..Service1",
		Methods: []*api.Method{
			{
				Name:         "get",
				ID:           "..Service1.get",
				InputTypeID:  ".google.protobuf.Empty",
				OutputTypeID: ".google.protobuf.Empty",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							Verb:         "GET",
							PathTemplate: api.NewPathTemplate().WithLiteral("resource"),
						},
					},
				},
			},
		},
	}
	model := api.NewTestAPI(
		[]*api.Message{},
		[]*api.Enum{},
		[]*api.Service{service0, service1})
	api.CrossReference(model)
	return model
}
