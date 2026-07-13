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
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

func TestGenerateService_APIVersion(t *testing.T) {
	for _, test := range []struct {
		name       string
		apiVersion string
		want       string
	}{
		{
			name:       "WithAPIVersion",
			apiVersion: "v1_20260713",
			want: `      let query = [
        URLQueryItem(name: "$alt", value: "json;enum-encoding=int"),
        URLQueryItem(name: "$apiVersion", value: "v1_20260713"),
      ]`,
		},
		{
			name:       "WithoutAPIVersion",
			apiVersion: "", // same as not-set
			want: `      let query = [
        URLQueryItem(name: "$alt", value: "json;enum-encoding=int"),
      ]`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()

			requestType := api.NewTestMessage("Request").
				WithFields(api.NewTestField("name").WithType(api.TypezString))
			responseType := api.NewTestMessage("Response")
			method := api.NewTestMethod("MethodWithVersion").
				WithInput(requestType).
				WithOutput(responseType).
				WithVerb("POST").
				WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("method"))
			method.APIVersion = test.apiVersion
			service := api.NewTestService("TestService").WithMethods(method)

			model := api.NewTestAPI([]*api.Message{requestType, responseType}, nil, []*api.Service{service})
			model.PackageName = "test"

			cfg := &parser.ModelConfig{
				Codec: map[string]string{
					"copyright-year": "2026",
				},
			}

			swiftCfg := swiftConfig(t, []config.SwiftDependency{
				{Name: "GoogleCloudGax", RequiredByServices: true},
			})

			if err := Generate(t.Context(), model, outDir, cfg, swiftCfg); err != nil {
				t.Fatal(err)
			}

			filename := filepath.Join(outDir, "Sources", "GoogleTest", "TestService+Stub.swift")
			content, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			contentStr := string(content)

			got := extractBlock(t, contentStr, "      let query = [", "\n      ]")
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
