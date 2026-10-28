// Copyright 2025 Google LLC
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

package discovery

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/api/apitest"
)

func TestMaybeInlineObject(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "package"
	input := &schema{
		Type:        "object",
		Description: "A field with an inline object.",
		Deprecated:  true,
		Properties: []*property{
			{
				Name: "stringField",
				Schema: &schema{
					Type:        "string",
					Description: "The stringField field.",
				},
			},
			{
				Name: "intField",
				Schema: &schema{
					Type:        "string",
					Format:      "uint64",
					Description: "The intField field.",
				},
			},
		},
	}
	message := api.NewTestMessage("Message").WithPackage("package")
	field, err := maybeInlineObjectField(model, message, "inline", input)
	if err != nil {
		t.Fatal(err)
	}

	wantField := api.NewTestMessage("Message").
		WithPackage("package").
		WithFields(
			api.NewTestField("inline").
				WithDocumentation("A field with an inline object.").
				WithDeprecated(true).
				WithOptional().
				WithType(api.TypezMessage).
				WithTypezID(".package.Message.inline"),
		).Fields[0]
	wantField.Parent = nil
	if diff := cmp.Diff(wantField, field); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantInlineMessage := api.NewTestMessage("inline").
		WithPackage("package").
		WithID(".package.Message.inline").
		WithDocumentation("The message type for the [inline][package.Message.inline] field.").
		WithFields(
			api.NewTestField("stringField").
				WithDocumentation("The stringField field.").
				WithType(api.TypezString).
				WithTypezID("string").
				WithOptional(),
			api.NewTestField("intField").
				WithDocumentation("The intField field.").
				WithType(api.TypezUint64).
				WithTypezID("uint64").
				WithOptional(),
		)
	for _, f := range wantInlineMessage.Fields {
		f.Parent = nil
	}
	wantInlineMessage.Parent = message
	gotInlineMessage := model.Message(wantInlineMessage.ID)
	if gotInlineMessage == nil {
		t.Fatalf("missing inline message %s", wantInlineMessage.ID)
	}
	apitest.CheckMessage(t, gotInlineMessage, wantInlineMessage)
	if gotInlineMessage.Parent != message {
		t.Errorf("mismatched parent in inline message, got=%v, want=%v", gotInlineMessage.Parent, message)
	}
}

func TestArrayWithInlineObject(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "package"
	input := &property{
		Name: "arrayWithObject",
		Schema: &schema{
			Type:        "array",
			Description: "An array field with an inline object.",
			ItemSchema: &schema{
				Type: "object",
				Properties: []*property{
					{
						Name: "stringField",
						Schema: &schema{
							Type:        "string",
							Description: "The stringField field.",
						},
					},
					{
						Name: "intField",
						Schema: &schema{
							Type:        "string",
							Format:      "uint64",
							Description: "The intField field.",
						},
					},
				},
			},
		},
	}
	message := api.NewTestMessage("Message").WithPackage("package")
	field, err := makeArrayField(model, message, input)
	if err != nil {
		t.Fatal(err)
	}

	wantField := api.NewTestMessage("Message").
		WithPackage("package").
		WithFields(
			api.NewTestField("arrayWithObject").
				WithDocumentation("An array field with an inline object.").
				WithRepeated().
				WithType(api.TypezMessage).
				WithTypezID(".package.Message.arrayWithObject"),
		).Fields[0]
	wantField.Parent = nil
	if diff := cmp.Diff(wantField, field); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantInlineMessage := api.NewTestMessage("arrayWithObject").
		WithPackage("package").
		WithID(".package.Message.arrayWithObject").
		WithDocumentation("The message type for the [arrayWithObject][package.Message.arrayWithObject] field.").
		WithFields(
			api.NewTestField("stringField").
				WithDocumentation("The stringField field.").
				WithType(api.TypezString).
				WithTypezID("string").
				WithOptional(),
			api.NewTestField("intField").
				WithDocumentation("The intField field.").
				WithType(api.TypezUint64).
				WithTypezID("uint64").
				WithOptional(),
		)
	for _, f := range wantInlineMessage.Fields {
		f.Parent = nil
	}
	wantInlineMessage.Parent = message
	gotInlineMessage := model.Message(wantInlineMessage.ID)
	if gotInlineMessage == nil {
		t.Fatalf("missing inline message %s", wantInlineMessage.ID)
	}
	apitest.CheckMessage(t, gotInlineMessage, wantInlineMessage)
	if gotInlineMessage.Parent != message {
		t.Errorf("mismatched parent in inline message, got=%v, want=%v", gotInlineMessage.Parent, message)
	}
}

func TestMaybeInlineObjectErrors(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "package"
	input := &schema{
		Type:        "object",
		Description: "A field with an inline object.",
		Properties: []*property{
			{
				Name: "badField",
				Schema: &schema{
					Type:   "string",
					Format: "--invalid--",
				},
			},
		},
	}
	message := api.NewTestMessage("Message").WithPackage("package")
	if field, err := maybeInlineObjectField(model, message, "inline", input); err == nil {
		t.Errorf("expected an error with an invalid inline object, got=%v", field)
	}
}

func TestArrayWithInlineObjectError(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "package"
	input := &property{
		Name: "arrayWithObject",
		Schema: &schema{
			Type:        "array",
			Description: "An array field with an inline object.",
			ItemSchema: &schema{
				Type: "object",
				Properties: []*property{
					{
						Name: "stringField",
						Schema: &schema{
							Type:        "string",
							Format:      "--invalid--",
							Description: "The stringField field.",
						},
					},
				},
			},
		},
	}
	message := api.NewTestMessage("Message").WithPackage("package")
	if field, err := makeArrayField(model, message, input); err == nil {
		t.Errorf("expected an error with an invalid inline object, got=%v", field)
	}
}
