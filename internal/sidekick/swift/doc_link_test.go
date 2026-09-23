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

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestDocLink(t *testing.T) {
	enumValue := api.NewTestEnumValue("ENUM_VALUE", 0)
	someEnum := api.NewTestEnum("SomeEnum").WithValues(enumValue)
	response := api.NewTestField("response")
	errorz := api.NewTestField("error")
	result := api.NewTestOneOf("result").WithFields(response, errorz)
	someMessage := api.NewTestMessage("SomeMessage").
		WithPackage("test.v1").
		WithFields(
			api.NewTestField("unused"),
			api.NewTestField("field"),
			api.NewTestField("typez"),
		).
		WithOneOfs(result).
		WithEnums(someEnum)
	otherMessage := api.NewTestMessage("OtherMessage").
		WithPackage("other.v1")
	someService := api.NewTestService("SomeService").
		WithPackage("test.v1").
		WithMethods(
			api.NewTestMethod("CreateFoo"),
		)

	model := api.NewTestAPI(
		[]*api.Message{otherMessage, someMessage},
		[]*api.Enum{someEnum},
		[]*api.Service{someService})

	c := newTestCodec(t, model, nil)
	c.withExtraDependencies(t, []config.SwiftDependency{
		{Name: "OtherPrefix", ApiPackage: "other.v1"},
	})

	for _, test := range []struct {
		name   string
		link   string
		scopes []string
		want   string
	}{
		{
			name:   "message link",
			link:   "SomeMessage",
			scopes: []string{"test.v1"},
			want:   "<doc:SomeMessage>",
		},
		{
			name:   "enum link",
			link:   "SomeMessage.SomeEnum",
			scopes: []string{"test.v1"},
			want:   "<doc:SomeMessage/SomeEnum>",
		},
		{
			name:   "field link",
			link:   "SomeMessage.field",
			scopes: []string{"test.v1"},
			want:   "<doc:SomeMessage/field>",
		},
		{
			name:   "oneof field link (1)",
			link:   "SomeMessage.error",
			scopes: []string{"test.v1"},
			want:   "<doc:SomeMessage/OneOf_Result/error(_:)>",
		},
		{
			name:   "oneof field link (2)",
			link:   "SomeMessage.response",
			scopes: []string{"test.v1"},
			want:   "<doc:SomeMessage/OneOf_Result/response(_:)>",
		},
		{
			name:   "enum value link",
			link:   "SomeMessage.SomeEnum.ENUM_VALUE",
			scopes: []string{"test.v1"},
			want:   "<doc:SomeMessage/SomeEnum/enumValue>",
		},
		{
			name:   "method link",
			link:   "SomeService.CreateFoo",
			scopes: []string{"test.v1"},
			want:   "<doc:SomeServiceClient/createFoo(request:options:)>",
		},
		{
			name:   "service link",
			link:   "SomeService",
			scopes: []string{"test.v1"},
			want:   "<doc:SomeServiceClient>",
		},
		{
			name:   "fully qualified message link",
			link:   "test.v1.SomeMessage",
			scopes: []string{},
			want:   "<doc:SomeMessage>",
		},
		{
			name:   "different package link",
			link:   "other.v1.OtherMessage",
			scopes: []string{},
			want:   "https://www.google.com/search?q=Swift+other.v1+OtherPrefix.OtherMessage",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := c.linkDefinition(test.link, test.scopes)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Errorf("docLink() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestDocLinkAmbiguity(t *testing.T) {
	globalAmbiguous2 := api.NewTestMessage("Ambiguous2").
		WithPackage("test.v1")
	nestedAmbiguous1 := api.NewTestMessage("Ambiguous1").
		WithPackage("test.v1").
		WithID(".test.v1.Parent.Ambiguous1")
	nestedAmbiguous2 := api.NewTestMessage("Ambiguous2").
		WithPackage("test.v1").
		WithID(".test.v1.Parent.Ambiguous2").
		WithFields(api.NewTestField("field_name"))
	parent := api.NewTestMessage("Parent").
		WithPackage("test.v1")

	model := api.NewTestAPI(
		[]*api.Message{globalAmbiguous2, parent, nestedAmbiguous1, nestedAmbiguous2},
		nil,
		nil)

	c := newTestCodec(t, model, nil)

	tests := []struct {
		name   string
		link   string
		scopes []string
		want   string
	}{
		{
			name:   "resolve to sibling inside parent",
			link:   "Ambiguous2.field_name",
			scopes: []string{"test.v1.Parent", "test.v1"},
			want:   "<doc:Parent/Ambiguous2/fieldName>",
		},
		{
			name:   "resolve to global outside parent",
			link:   "Ambiguous2",
			scopes: []string{"test.v1"},
			want:   "<doc:Ambiguous2>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.linkDefinition(tt.link, tt.scopes)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("docLink() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDocLink_NameOverrides(t *testing.T) {
	service := api.NewTestService("Storage").
		WithPackage("google.storage.v2").
		WithMethods(
			api.NewTestMethod("CreateBucket"),
		)

	model := api.NewTestAPI(nil, nil, []*api.Service{service})
	library := &config.Library{
		Swift: &config.SwiftPackage{
			NameOverrides: map[string]string{
				".google.storage.v2.Storage": "StorageAdmin",
			},
		},
	}

	c, err := newCodec(model, library, nil, ".")
	if err != nil {
		t.Fatal(err)
	}

	gotService := c.serviceDocLink(service)
	wantService := "<doc:StorageAdminClient>"
	if gotService != wantService {
		t.Errorf("serviceDocLink() = %q, want %q", gotService, wantService)
	}

	gotMethod, err := c.methodDocLink(service.Methods[0])
	if err != nil {
		t.Fatal(err)
	}
	wantMethod := "<doc:StorageAdminClient/createBucket(request:options:)>"
	if gotMethod != wantMethod {
		t.Errorf("methodDocLink() = %q, want %q", gotMethod, wantMethod)
	}
}

func TestDocLink_ModuleNameOverrides(t *testing.T) {
	service := api.NewTestService("Storage").
		WithPackage("google.storage.v2").
		WithMethods(
			api.NewTestMethod("CreateBucket"),
		)

	model := api.NewTestAPI(nil, nil, []*api.Service{service})
	library := &config.Library{
		Swift: &config.SwiftPackage{
			NameOverrides: map[string]string{
				".google.storage.v2.Storage": "PackageDefault",
			},
		},
	}
	module := &config.SwiftModule{
		NameOverrides: map[string]string{
			".google.storage.v2.Storage": "ModuleOverride",
		},
	}

	c, err := newCodec(model, library, module, ".")
	if err != nil {
		t.Fatal(err)
	}

	gotService := c.serviceDocLink(service)
	wantService := "<doc:ModuleOverrideClient>"
	if gotService != wantService {
		t.Errorf("serviceDocLink() = %q, want %q", gotService, wantService)
	}
}
