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
		name       string
		doc        string
		setupModel func() *api.API
		wantLines  []string
		wantHas    bool
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
	} {
		t.Run(test.name, func(t *testing.T) {
			svc := api.NewTestService("SecretManagerService")
			svc.Documentation = test.doc
			var model *api.API
			if test.setupModel != nil {
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
