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

package rust

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	libconfig "github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

func TestGenerateExplicitEmptyRouting(t *testing.T) {
	for _, test := range []struct {
		name     string
		filePath string
		codec    map[string]string
	}{
		{
			name:     "crate template",
			filePath: filepath.Join("src", "transport.rs"),
			codec: map[string]string{
				"package:wkt":       "source=google.protobuf,package=google-cloud-wkt",
				"default-transport": "grpc",
				"routing-required":  "true",
			},
		},
		{
			name:     "grpc-client template",
			filePath: "transport.rs",
			codec: map[string]string{
				"template-override": "templates/grpc-client",
				"package:wkt":       "source=google.protobuf,package=google-cloud-wkt",
				"routing-required":  "true",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := api.NewTestMessage("Request").
				WithPackage("test.v1").
				WithFields(
					api.NewTestField("table_name").WithType(api.TypezString),
				)
			response := api.NewTestMessage("Response").WithPackage("test.v1")

			explicitEmptyUnary := api.NewTestMethod("ExplicitEmpty").
				WithInput(request).
				WithOutput(response).
				WithVerb("GET").
				WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").WithVariableNamed("table_name").Build()).
				WithRouting()

			explicitEmptyStreaming := api.NewTestMethod("ExplicitEmptyStream").
				WithInput(request).
				WithOutput(response).
				WithServerSideStreaming().
				WithVerb("GET").
				WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").WithVariableNamed("table_name").Build()).
				WithRouting()

			service := api.NewTestService("TestService").
				WithPackage("test.v1").
				WithMethods(explicitEmptyUnary, explicitEmptyStreaming)

			model := api.NewTestAPI([]*api.Message{request, response}, nil, []*api.Service{service}).
				WithPackageName("test.v1")
			if err := api.CrossReference(model); err != nil {
				t.Fatal(err)
			}

			outDir := t.TempDir()
			cfg := &parser.ModelConfig{
				SpecificationFormat: libconfig.SpecProtobuf,
				Codec:               test.codec,
			}
			if err := Generate(t.Context(), model, outDir, cfg); err != nil {
				t.Fatal(err)
			}

			contentBytes, err := os.ReadFile(filepath.Join(outDir, test.filePath))
			if err != nil {
				t.Fatal(err)
			}
			content := string(contentBytes)

			if got := strings.Count(content, "let x_goog_request_params = String::new();"); got != 2 {
				t.Errorf("expected 2 empty routing header bindings, got %d in:\n%s", got, content)
			}
			for _, forbidden := range []string{
				"table_name={v}",
				"BindingError",
				"use gaxi::routing_parameter::Segment;",
			} {
				if strings.Contains(content, forbidden) {
					t.Errorf("expected generated %s not to contain %q in:\n%s", test.filePath, forbidden, content)
				}
			}
		})
	}
}
