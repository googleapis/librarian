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
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

func TestGenerateSnippets(t *testing.T) {
	for _, test := range []struct {
		name     string
		repeated bool
		file     string
		want     string
	}{
		{
			name:     "quickstart singular",
			repeated: false,
			file:     "TestServiceQuickstart.swift",
			want: `func sample(name: String, ) async throws {
  let client = try GoogleTest.TestServiceClient()
  let response = try await client.getThing(
    request: GetThingRequest()
  .with {
    $0.name = "\(name)"
  }
)
  print("Success: \(response)")
}`,
		},
		{
			name:     "quickstart repeated",
			repeated: true,
			file:     "TestServiceQuickstart.swift",
			want: `func sample(name: String, ) async throws {
  let client = try GoogleTest.TestServiceClient()
  let response = try await client.getThing(
    request: GetThingRequest()
  .with {
    $0.name = ["\(name)"]
  }
)
  print("Success: \(response)")
}`,
		},
		{
			name:     "method snippet singular",
			repeated: false,
			file:     "TestService_GetThing.swift",
			want: `func sample(client: TestServiceClient, name: String) async throws {
  let response = try await client.getThing(
    request: GetThingRequest()
  .with {
    $0.name = "\(name)"
  }
)
  print("Success: \(response)")
}`,
		},
		{
			name:     "method snippet repeated",
			repeated: true,
			file:     "TestService_GetThing.swift",
			want: `func sample(client: TestServiceClient, name: String) async throws {
  let response = try await client.getThing(
    request: GetThingRequest()
  .with {
    $0.name = ["\(name)"]
  }
)
  print("Success: \(response)")
}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()
			thingResource := api.NewTestResource("test.googleapis.com/Thing").
				WithSingular("thing").
				WithPlural("things")
			thing := api.NewTestMessage("Thing").WithResource(thingResource).WithFields(
				api.NewTestField("name").WithType(api.TypezString).WithResourceReference(
					thingResource.Type,
				),
				api.NewTestField("cool_attribute").WithType(api.TypezBytes),
			)
			name := api.NewTestField("name").
				WithType(api.TypezString).
				WithResourceReference(thingResource.Type)
			if test.repeated {
				name = name.WithRepeated()
			}
			getThingRequest := api.NewTestMessage("GetThingRequest").
				WithFields(name)
			getThing := api.NewTestMethod("GetThing").
				WithInput(getThingRequest).
				WithOutput(thing).
				WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("one"))
			testService := api.NewTestService("TestService").WithMethods(getThing)
			model := api.NewTestAPI([]*api.Message{thing, getThingRequest}, nil, []*api.Service{testService})
			model.PackageName = "test"
			model.AddResource(thingResource)
			if err := api.CrossReference(model); err != nil {
				t.Fatal(err)
			}
			cfg := &parser.ModelConfig{}
			if err := Generate(t.Context(), model, outDir, cfg, swiftConfig(t, nil)); err != nil {
				t.Fatal(err)
			}
			contentsBytes, err := os.ReadFile(filepath.Join(outDir, "Snippets", test.file))
			if err != nil {
				t.Fatal(err)
			}
			contents := string(contentsBytes)
			got := extractBlock(t, contents, "func sample(", "\n}")
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
