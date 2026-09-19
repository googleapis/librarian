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

func TestAnnotateService(t *testing.T) {
	for _, test := range []struct {
		name       string
		service    *api.Service
		sourceFile string
		libCfg     *config.CppLibrary
		want       *serviceAnnotations
	}{
		{
			name:    "simple service without definition location",
			service: api.NewTestService("SimpleService"),
			want: &serviceAnnotations{
				Name:          "SimpleService",
				CopyrightYear: "2026",
				BoilerPlate:   license.HeaderBulk(),
				Model: &modelAnnotations{
					CopyrightYear: "2026",
					BoilerPlate:   license.HeaderBulk(),
				},
				SourceFile: "",
			},
		},
		{
			name:       "service with definition location",
			service:    api.NewTestService("EchoService"),
			sourceFile: "google/example/echo.proto",
			want: &serviceAnnotations{
				Name:          "EchoService",
				CopyrightYear: "2026",
				BoilerPlate:   license.HeaderBulk(),
				Model: &modelAnnotations{
					CopyrightYear: "2026",
					BoilerPlate:   license.HeaderBulk(),
				},
				SourceFile: "google/example/echo.proto",
			},
		},
		{
			name:       "service with custom copyright year",
			service:    api.NewTestService("CustomYearService"),
			sourceFile: "google/example/custom.proto",
			libCfg: &config.CppLibrary{
				InitialCopyrightYear: "2024",
			},
			want: &serviceAnnotations{
				Name:          "CustomYearService",
				CopyrightYear: "2024",
				BoilerPlate:   license.HeaderBulk(),
				Model: &modelAnnotations{
					CopyrightYear: "2024",
					BoilerPlate:   license.HeaderBulk(),
				},
				SourceFile: "google/example/custom.proto",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			model := api.NewTestAPI(nil, nil, []*api.Service{test.service})
			if test.sourceFile != "" {
				model.DefinitionLocations = map[string]api.SourceLocation{
					test.service.ID: {Filename: test.sourceFile, Line: 42},
				}
			}
			c := newCodec(test.libCfg)
			if err := c.annotateModel(model); err != nil {
				t.Fatal(err)
			}
			got, ok := test.service.Codec.(*serviceAnnotations)
			if !ok {
				t.Fatalf("expected *serviceAnnotations, got %T", test.service.Codec)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
