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
	enumValue := &api.EnumValue{
		Name: "ENUM_VALUE",
		ID:   ".test.v1.SomeMessage.SomeEnum.ENUM_VALUE",
	}
	someEnum := &api.Enum{
		Name:    "SomeEnum",
		ID:      ".test.v1.SomeMessage.SomeEnum",
		Values:  []*api.EnumValue{enumValue},
		Package: "test.v1",
	}
	enumValue.Parent = someEnum
	response := &api.Field{
		Name:    "response",
		ID:      ".test.v1.SomeMessage.response",
		IsOneOf: true,
	}
	errorz := &api.Field{
		Name:    "error",
		ID:      ".test.v1.SomeMessage.error",
		IsOneOf: true,
	}
	typez := &api.Field{
		Name: "type",
		ID:   ".test.v1.SomeMessage.type",
	}
	someMessage := &api.Message{
		Name:    "SomeMessage",
		ID:      ".test.v1.SomeMessage",
		Package: "test.v1",
		Enums:   []*api.Enum{someEnum},
		Fields: []*api.Field{
			{Name: "unused"}, {Name: "field"}, response, errorz, typez,
		},
		OneOfs: []*api.OneOf{
			{
				Name:   "result",
				ID:     ".test.v1.SomeMessage.result",
				Fields: []*api.Field{response, errorz},
			},
		},
	}
	otherMessage := &api.Message{
		Name:    "OtherMessage",
		ID:      ".other.v1.OtherMessage",
		Package: "other.v1",
	}
	someService := &api.Service{
		Name:    "SomeService",
		ID:      ".test.v1.SomeService",
		Package: "test.v1",
		Methods: []*api.Method{
			{
				Name: "CreateFoo",
				ID:   ".test.v1.SomeService.CreateFoo",
			},
		},
	}

	model := api.NewTestAPI(
		[]*api.Message{otherMessage, someMessage},
		[]*api.Enum{someEnum},
		[]*api.Service{someService})

	c := newTestCodec(t, model, nil)
	c.withExtraDependencies(t, []config.SwiftDependency{
		{Name: "OtherPrefix", ApiPackage: "other.v1"},
	})

	tests := []struct {
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
			name:   "enum value link",
			link:   "SomeMessage.SomeEnum.ENUM_VALUE",
			scopes: []string{"test.v1"},
			want:   "<doc:SomeMessage/SomeEnum/enumValue>",
		},
		{
			name:   "method link",
			link:   "SomeService.CreateFoo",
			scopes: []string{"test.v1"},
			want:   "<doc:SomeService/createFoo(request:)>",
		},
		{
			name:   "service link",
			link:   "SomeService",
			scopes: []string{"test.v1"},
			want:   "<doc:SomeService>",
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

func TestDocLinkAmbiguity(t *testing.T) {
	globalAmbiguous2 := &api.Message{
		Name:    "Ambiguous2",
		ID:      ".test.v1.Ambiguous2",
		Package: "test.v1",
	}
	nestedAmbiguous1 := &api.Message{
		Name:    "Ambiguous1",
		ID:      ".test.v1.Parent.Ambiguous1",
		Package: "test.v1",
	}
	nestedAmbiguous2 := &api.Message{
		Name:    "Ambiguous2",
		ID:      ".test.v1.Parent.Ambiguous2",
		Package: "test.v1",
		Fields: []*api.Field{
			{Name: "field_name"},
		},
	}
	parent := &api.Message{
		Name:     "Parent",
		ID:       ".test.v1.Parent",
		Package:  "test.v1",
		Messages: []*api.Message{nestedAmbiguous1, nestedAmbiguous2},
	}
	nestedAmbiguous1.Parent = parent
	nestedAmbiguous2.Parent = parent

	model := api.NewTestAPI(
		[]*api.Message{globalAmbiguous2, parent, nestedAmbiguous1, nestedAmbiguous2},
		[]*api.Enum{},
		[]*api.Service{})

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
