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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
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
  let client = try Test.TestServiceClient()
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
  let client = try Test.TestServiceClient()
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
			library := &config.Library{
				Swift: swiftConfig(t, nil),
			}
			if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
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

// TestGenerateSnippets_Diagnose covers the generated snippets.
//
// A snippet is a standalone executable, so nothing in it is ever marked
// deprecated. Unlike the public client, it needs the attribute even when the
// method or the service it exercises is deprecated.
func TestGenerateSnippets_Diagnose(t *testing.T) {
	for _, test := range []struct {
		name              string
		serviceDeprecated bool
		methodDeprecated  bool
		fieldDeprecated   bool
		wantSample        bool
		wantRunner        bool
	}{
		{
			name:              "deprecated-service",
			serviceDeprecated: true,
			wantSample:        true,
			wantRunner:        true,
		},
		{
			name:             "deprecated-method",
			methodDeprecated: true,
			wantSample:       true,
		},
		{
			name:            "deprecated-request-field",
			fieldDeprecated: true,
			wantSample:      true,
		},
		{
			name: "not-deprecated",
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
			)
			getThingRequest := api.NewTestMessage("GetThingRequest").WithFields(
				api.NewTestField("name").
					WithType(api.TypezString).
					WithResourceReference(thingResource.Type).
					WithDeprecated(test.fieldDeprecated),
			)
			getThing := api.NewTestMethod("GetThing").
				WithInput(getThingRequest).
				WithOutput(thing).
				WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("one")).
				WithDeprecated(test.methodDeprecated)
			testService := api.NewTestService("TestService").
				WithDeprecated(test.serviceDeprecated).
				WithMethods(getThing)
			model := api.NewTestAPI([]*api.Message{thing, getThingRequest}, nil, []*api.Service{testService})
			model.PackageName = "test"
			model.AddResource(thingResource)
			if err := api.CrossReference(model); err != nil {
				t.Fatal(err)
			}
			library := &config.Library{
				Swift: swiftConfig(t, nil),
			}
			if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
				t.Fatal(err)
			}

			read := func(basename string) string {
				t.Helper()
				contents, err := os.ReadFile(filepath.Join(outDir, "Snippets", basename))
				if err != nil {
					t.Fatal(err)
				}
				return string(contents)
			}

			// The method snippet names the client type, the method and the
			// request fields; the runner's `main()` only names the client type.
			methodSnippet := read("TestService_GetThing.swift")
			checkDiagnose(t, methodSnippet, "", "func sample(", test.wantSample)
			checkDiagnose(t, methodSnippet, "    ", "static func main()", test.wantRunner)

			// The client snippet inlines the quickstart method's body, so it
			// names everything the method snippet does.
			checkDiagnose(t, read("TestServiceQuickstart.swift"), "", "func sample(", test.wantSample)
		})
	}
}

func TestGenerateSnippets_UpdateMask(t *testing.T) {
	outDir := t.TempDir()
	thingResource := api.NewTestResource("test.googleapis.com/Thing").
		WithSingular("thing").
		WithPlural("things")
	thing := api.NewTestMessage("Thing").WithResource(thingResource).WithFields(
		api.NewTestField("name").WithType(api.TypezString).WithResourceReference(
			thingResource.Type,
		),
	)
	updateMask := api.NewTestField("update_mask").WithType(api.TypezMessage).WithTypezID(api.WktFieldMaskID)
	updateThingRequest := api.NewTestMessage("UpdateThingRequest").WithFields(
		api.NewTestField("name").WithType(api.TypezString).WithResourceReference(thingResource.Type),
		updateMask,
	)
	updateThing := api.NewTestMethod("UpdateThing").
		WithInput(updateThingRequest).
		WithOutput(thing).
		WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("things"))
	testService := api.NewTestService("TestService").WithMethods(updateThing)
	model := api.NewTestAPI([]*api.Message{thing, updateThingRequest}, nil, []*api.Service{testService})
	model.PackageName = "test"
	model.AddResource(thingResource)
	model.LoadWellKnownTypes()
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}
	updateThing.SampleInfo = &api.SampleInfo{
		UpdateMaskField: updateMask,
	}
	library := &config.Library{
		Swift: swiftConfig(t, nil),
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}
	contentsBytes, err := os.ReadFile(filepath.Join(outDir, "Snippets", "TestService_UpdateThing.swift"))
	if err != nil {
		t.Fatal(err)
	}
	contents := string(contentsBytes)
	if !strings.Contains(contents, "import GoogleWKT") {
		t.Errorf("expected snippet to contain 'import GoogleWKT', got:\n%s", contents)
	}
	if !strings.Contains(contents, "$0.updateMask = GoogleWKT.FieldMask(paths: [\"field.path1\", \"field.path2\"])") {
		t.Errorf("expected snippet to contain updateMask assignment, got:\n%s", contents)
	}
}
