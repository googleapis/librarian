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
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateService(t *testing.T) {
	for _, test := range []struct {
		name        string
		serviceName string
		doc         string
		wantName    string
		wantDocs    []string
	}{
		{
			name:        "IAM service",
			serviceName: "IAM",
			doc:         "IAM service documentation.",
			wantName:    "IAM",
			wantDocs:    []string{"IAM service documentation."},
		},
		{
			name:        "SecretManagerService",
			serviceName: "SecretManagerService",
			doc:         "Secret Manager Service documentation.\nLine 2.",
			wantName:    "SecretManagerService",
			wantDocs:    []string{"Secret Manager Service documentation.", "Line 2."},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := &api.Service{
				Name:          test.serviceName,
				Documentation: test.doc,
			}
			model := api.NewTestAPI(nil, nil, []*api.Service{s})
			codec := newTestCodec(t, model, nil)

			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}

			want := &serviceAnnotations{
				Name:     test.wantName,
				DocLines: test.wantDocs,
			}

			if diff := cmp.Diff(want, s.Codec, cmpopts.IgnoreFields(serviceAnnotations{}, "BoilerPlate", "CopyrightYear", "PackageName", "QuickstartMethod")); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateService_SkipNoBindings(t *testing.T) {
	inputType := &api.Message{
		Name:    "Request",
		ID:      ".test.Request",
		Package: "test",
	}
	outputType := &api.Message{
		Name:    "Response",
		ID:      ".test.Response",
		Package: "test",
	}
	service := &api.Service{
		Name:    "TestService",
		ID:      ".test.TestService",
		Package: "test",
		Methods: []*api.Method{
			{
				Name:         "ValidMethod",
				InputTypeID:  inputType.ID,
				InputType:    inputType,
				OutputTypeID: outputType.ID,
				OutputType:   outputType,
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{{Verb: "GET", PathTemplate: &api.PathTemplate{}}},
				},
			},
			{
				Name:         "NoBindingMethod",
				InputTypeID:  inputType.ID,
				InputType:    inputType,
				OutputTypeID: outputType.ID,
				OutputType:   outputType,
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{},
				},
			},
			{
				Name:         "NilPathInfoMethod",
				InputTypeID:  inputType.ID,
				InputType:    inputType,
				OutputTypeID: outputType.ID,
				OutputType:   outputType,
				PathInfo:     nil,
			},
		},
	}

	model := api.NewTestAPI(nil, nil, []*api.Service{service})
	codec := newTestCodec(t, model, nil)
	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	serviceCodec := service.Codec.(*serviceAnnotations)
	var gotNames []string
	for _, m := range serviceCodec.RestMethods {
		gotNames = append(gotNames, m.Name)
	}
	wantNames := []string{"ValidMethod"}
	if diff := cmp.Diff(wantNames, gotNames); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateService_Quickstart(t *testing.T) {
	for _, test := range []struct {
		name             string
		quickstartMethod *api.Method
		wantQuickstart   bool
	}{
		{
			name:             "nil quickstart",
			quickstartMethod: nil,
			wantQuickstart:   false,
		},
		{
			name: "non-generated quickstart (nil PathInfo)",
			quickstartMethod: &api.Method{
				Name:     "Quickstart",
				PathInfo: nil,
			},
			wantQuickstart: false,
		},
		{
			name: "non-generated quickstart (empty bindings)",
			quickstartMethod: &api.Method{
				Name: "Quickstart",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{},
				},
			},
			wantQuickstart: false,
		},
		{
			name: "generated quickstart",
			quickstartMethod: &api.Method{
				Name: "Quickstart",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{{Verb: "GET", PathTemplate: &api.PathTemplate{}}},
				},
			},
			wantQuickstart: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := &api.Service{
				Name:             "TestService",
				QuickstartMethod: test.quickstartMethod,
			}

			model := api.NewTestAPI(nil, nil, []*api.Service{service})
			codec := newTestCodec(t, model, nil)
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}

			annotations, ok := service.Codec.(*serviceAnnotations)
			if !ok {
				t.Fatal("service.Codec is not *serviceAnnotations")
			}

			if test.wantQuickstart {
				if annotations.QuickstartMethod == nil {
					t.Error("expected QuickstartMethod to be set, got nil")
				}
			} else {
				if annotations.QuickstartMethod != nil {
					t.Errorf("expected QuickstartMethod to be nil, got %v", annotations.QuickstartMethod)
				}
			}
		})
	}
}
