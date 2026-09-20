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

package cpp

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/license"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateService_ClientCommentReferences(t *testing.T) {
	for _, test := range []struct {
		name                  string
		doc                   string
		setupService          func() *api.Service
		setupModel            func() *api.API
		setupModelWithService func(svc *api.Service) *api.API
		wantLines             []string
		wantHas               bool
	}{
		{
			name: "service documentation with references",
			doc:  "Service managing secrets.\n* [Secret][google.cloud.secretmanager.v1.Secret]\n* [SecretVersion][google.cloud.secretmanager.v1.SecretVersion]",
			setupModel: func() *api.API {
				model := api.NewTestAPI(nil, nil, nil)
				model.AddDefinitionLocation("google.cloud.secretmanager.v1.Secret", api.SourceLocation{
					Filename: "google/cloud/secretmanager/v1/resources.proto",
					Line:     39,
				})
				model.AddDefinitionLocation("google.cloud.secretmanager.v1.SecretVersion", api.SourceLocation{
					Filename: "google/cloud/secretmanager/v1/resources.proto",
					Line:     181,
				})
				return model
			},
			wantLines: []string{
				"[google.cloud.secretmanager.v1.Secret]: @googleapis_reference_link{google/cloud/secretmanager/v1/resources.proto#L39}",
				"[google.cloud.secretmanager.v1.SecretVersion]: @googleapis_reference_link{google/cloud/secretmanager/v1/resources.proto#L181}",
			},
			wantHas: true,
		},
		{
			name: "references are sorted alphabetically and deduplicated",
			doc:  "Service managing secrets.\n* [SecretVersion][google.cloud.secretmanager.v1.SecretVersion]\n* [Secret][google.cloud.secretmanager.v1.Secret]\n* [AnotherSecret][google.cloud.secretmanager.v1.Secret]",
			setupModel: func() *api.API {
				model := api.NewTestAPI(nil, nil, nil)
				model.AddDefinitionLocation("google.cloud.secretmanager.v1.Secret", api.SourceLocation{
					Filename: "google/cloud/secretmanager/v1/resources.proto",
					Line:     39,
				})
				model.AddDefinitionLocation("google.cloud.secretmanager.v1.SecretVersion", api.SourceLocation{
					Filename: "google/cloud/secretmanager/v1/resources.proto",
					Line:     181,
				})
				return model
			},
			wantLines: []string{
				"[google.cloud.secretmanager.v1.Secret]: @googleapis_reference_link{google/cloud/secretmanager/v1/resources.proto#L39}",
				"[google.cloud.secretmanager.v1.SecretVersion]: @googleapis_reference_link{google/cloud/secretmanager/v1/resources.proto#L181}",
			},
			wantHas: true,
		},
		{
			name: "references not found in model are ignored",
			doc:  "Service with unknown references.\n* [Secret][google.cloud.secretmanager.v1.Secret]\n* [Unknown][google.cloud.secretmanager.v1.Unknown]",
			setupModel: func() *api.API {
				model := api.NewTestAPI(nil, nil, nil)
				model.AddDefinitionLocation("google.cloud.secretmanager.v1.Secret", api.SourceLocation{
					Filename: "google/cloud/secretmanager/v1/resources.proto",
					Line:     39,
				})
				return model
			},
			wantLines: []string{
				"[google.cloud.secretmanager.v1.Secret]: @googleapis_reference_link{google/cloud/secretmanager/v1/resources.proto#L39}",
			},
			wantHas: true,
		},
		{
			name: "all references unresolved",
			doc:  "Service with unknown references.\n* [Unknown][google.cloud.secretmanager.v1.Unknown]",
			setupModel: func() *api.API {
				return api.NewTestAPI(nil, nil, nil)
			},
			wantLines: nil,
			wantHas:   false,
		},
		{
			name:      "malformed markdown links are ignored",
			doc:       "Service documentation with [broken link] and [unclosed][bracket and ][no_dot] and [Secret][invalid format].",
			wantLines: nil,
			wantHas:   false,
		},
		{
			name: "service documentation without references",
			doc:  "Simple documentation without references.",
			setupModel: func() *api.API {
				return api.NewTestAPI(nil, nil, nil)
			},
			wantLines: nil,
			wantHas:   false,
		},
		{
			name:      "empty documentation",
			doc:       "",
			wantLines: nil,
			wantHas:   false,
		},
		{
			name: "unimported cross-service symbols are filtered out when svc has definition location and dependencies",
			doc:  "Service managing secrets.\n* [Secret][google.cloud.secretmanager.v1.Secret]\n* [CryptoKey][google.cloud.kms.v1.CryptoKey]",
			setupService: func() *api.Service {
				req := api.NewTestMessage("CreateSecretRequest").WithPackage("google.cloud.secretmanager.v1")
				resp := api.NewTestMessage("Secret").WithPackage("google.cloud.secretmanager.v1")
				method := api.NewTestMethod("CreateSecret").WithInput(req).WithOutput(resp)
				return api.NewTestService("SecretManagerService").
					WithPackage("google.cloud.secretmanager.v1").
					WithMethods(method)
			},
			setupModelWithService: func(svc *api.Service) *api.API {
				req := svc.Methods[0].InputType
				resp := svc.Methods[0].OutputType
				model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc})
				model.AddDefinitionLocation(svc.ID, api.SourceLocation{
					Filename: "google/cloud/secretmanager/v1/service.proto",
					Line:     20,
				})
				model.AddDefinitionLocation("google.cloud.secretmanager.v1.Secret", api.SourceLocation{
					Filename: "google/cloud/secretmanager/v1/resources.proto",
					Line:     39,
				})
				model.AddDefinitionLocation("google.cloud.kms.v1.CryptoKey", api.SourceLocation{
					Filename: "google/cloud/kms/v1/resources.proto",
					Line:     50,
				})
				return model
			},
			wantLines: []string{
				"[google.cloud.secretmanager.v1.Secret]: @googleapis_reference_link{google/cloud/secretmanager/v1/resources.proto#L39}",
			},
			wantHas: true,
		},
		{
			name: "symbols in the same file as service definition are kept",
			doc:  "Service managing secrets.\n* [SameFileMessage][google.cloud.secretmanager.v1.SameFileMessage]\n* [CryptoKey][google.cloud.kms.v1.CryptoKey]",
			setupService: func() *api.Service {
				return api.NewTestService("SecretManagerService").
					WithPackage("google.cloud.secretmanager.v1")
			},
			setupModelWithService: func(svc *api.Service) *api.API {
				model := api.NewTestAPI(nil, nil, []*api.Service{svc})
				model.AddDefinitionLocation(svc.ID, api.SourceLocation{
					Filename: "google/cloud/secretmanager/v1/service.proto",
					Line:     20,
				})
				model.AddDefinitionLocation("google.cloud.secretmanager.v1.SameFileMessage", api.SourceLocation{
					Filename: "google/cloud/secretmanager/v1/service.proto",
					Line:     100,
				})
				model.AddDefinitionLocation("google.cloud.kms.v1.CryptoKey", api.SourceLocation{
					Filename: "google/cloud/kms/v1/resources.proto",
					Line:     50,
				})
				return model
			},
			wantLines: []string{
				"[google.cloud.secretmanager.v1.SameFileMessage]: @googleapis_reference_link{google/cloud/secretmanager/v1/service.proto#L100}",
			},
			wantHas: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var svc *api.Service
			if test.setupService != nil {
				svc = test.setupService()
			} else {
				svc = api.NewTestService("SecretManagerService")
			}
			svc.Documentation = test.doc
			var model *api.API
			if test.setupModelWithService != nil {
				model = test.setupModelWithService(svc)
			} else if test.setupModel != nil {
				model = test.setupModel()
			} else {
				model = api.NewTestAPI(nil, nil, nil)
			}
			modelAnn := &modelAnnotations{
				CopyrightYear: "2026",
				BoilerPlate:   license.HeaderBulk(),
			}
			c := newCodec(&config.CppLibrary{})
			if err := c.annotateService(svc, modelAnn, model); err != nil {
				t.Fatal(err)
			}
			got := svc.Codec.(*serviceAnnotations)
			gotHas := got.HasClientCommentReferences
			gotLines := got.ClientCommentReferenceLines
			if diff := cmp.Diff(test.wantHas, gotHas); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantLines, gotLines); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestServiceDependencyFiles(t *testing.T) {
	t.Run("nil svc or model", func(t *testing.T) {
		if _, ok := serviceDependencyFiles(nil, nil); ok {
			t.Errorf("expected false for nil svc and model")
		}
		svc := api.NewTestService("TestService")
		if _, ok := serviceDependencyFiles(svc, nil); ok {
			t.Errorf("expected false for nil model")
		}
		model := api.NewTestAPI(nil, nil, nil)
		if _, ok := serviceDependencyFiles(nil, model); ok {
			t.Errorf("expected false for nil svc")
		}
	})

	t.Run("svc without definition location", func(t *testing.T) {
		svc := api.NewTestService("TestService")
		model := api.NewTestAPI(nil, nil, []*api.Service{svc})
		if _, ok := serviceDependencyFiles(svc, model); ok {
			t.Errorf("expected false when svc has no definition location")
		}
	})

	t.Run("svc with definition location and dependencies", func(t *testing.T) {
		req := api.NewTestMessage("Req").WithPackage("test")
		resp := api.NewTestMessage("Resp").WithPackage("test")
		svc := api.NewTestService("TestService").WithPackage("test").WithMethods(
			api.NewTestMethod("Method").WithInput(req).WithOutput(resp),
		)
		model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc})
		model.AddDefinitionLocation(svc.ID, api.SourceLocation{
			Filename: "test/service.proto",
			Line:     10,
		})
		model.AddDefinitionLocation(req.ID, api.SourceLocation{
			Filename: "test/req.proto",
			Line:     20,
		})

		depFiles, ok := serviceDependencyFiles(svc, model)
		if !ok {
			t.Fatalf("expected ok=true")
		}
		if !depFiles["test/service.proto"] {
			t.Errorf("expected test/service.proto in depFiles")
		}
		if !depFiles["test/req.proto"] {
			t.Errorf("expected test/req.proto in depFiles")
		}
		if depFiles["test/other.proto"] {
			t.Errorf("did not expect test/other.proto in depFiles")
		}
	})

	t.Run("svc with definition location when FindDependencies returns error", func(t *testing.T) {
		method := api.NewTestMethod("Method")
		method.InputTypeID = "unknown.Type"
		method.OutputTypeID = "unknown.Type"
		svc := api.NewTestService("TestService").WithPackage("test").WithMethods(method)
		model := api.NewTestAPI(nil, nil, []*api.Service{svc})
		model.AddDefinitionLocation(svc.ID, api.SourceLocation{
			Filename: "test/service.proto",
			Line:     10,
		})

		depFiles, ok := serviceDependencyFiles(svc, model)
		if !ok {
			t.Fatalf("expected ok=true even when FindDependencies returns error")
		}
		if !depFiles["test/service.proto"] {
			t.Errorf("expected test/service.proto in depFiles")
		}
		if len(depFiles) != 1 {
			t.Errorf("expected only 1 file in depFiles, got %d: %v", len(depFiles), depFiles)
		}
	})
}
