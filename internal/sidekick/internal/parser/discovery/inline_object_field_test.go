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
	"github.com/googleapis/librarian/internal/sidekick/internal/api"
	"github.com/googleapis/librarian/internal/sidekick/internal/api/apitest"
)

func TestMaybeInlineObject(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "package"
	input := &schema{
		Type:        "object",
		Description: "A field with an inline object.",
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
	message := &api.Message{ID: ".package.Message"}
	field, err := maybeInlineObjectField(model, message, "inline", input)
	if err != nil {
		t.Fatal(err)
	}

	wantField := &api.Field{
		Name:          "inline",
		JSONName:      "inline",
		ID:            ".package.Message.inline",
		Documentation: "A field with an inline object.",
		Optional:      true,
		Typez:         api.MESSAGE_TYPE,
		TypezID:       ".package.Message.inline",
	}
	if diff := cmp.Diff(wantField, field); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}

	wantInlineMessage := &api.Message{
		ID:            ".package.Message.inline",
		Name:          "inline",
		Documentation: "The message type for the [inline][package.Message.inline] field.",
		Fields: []*api.Field{
			{
				Name:          "stringField",
				JSONName:      "stringField",
				ID:            ".package.Message.inline.stringField",
				Documentation: "The stringField field.",
				Typez:         api.STRING_TYPE,
				TypezID:       "string",
			},
			{
				Name:          "intField",
				JSONName:      "intField",
				ID:            ".package.Message.inline.intField",
				Documentation: "The intField field.",
				Typez:         api.UINT64_TYPE,
				TypezID:       "uint64",
			},
		},
		Parent: message,
	}
	gotInlineMessage, ok := model.State.MessageByID[wantInlineMessage.ID]
	if !ok {
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
	message := &api.Message{ID: ".package.Message"}
	field, err := makeArrayField(model, message, input)
	if err != nil {
		t.Fatal(err)
	}

	wantField := &api.Field{
		Name:          "arrayWithObject",
		JSONName:      "arrayWithObject",
		ID:            ".package.Message.arrayWithObject",
		Documentation: "An array field with an inline object.",
		Repeated:      true,
		Typez:         api.MESSAGE_TYPE,
		TypezID:       ".package.Message.arrayWithObject",
	}
	if diff := cmp.Diff(wantField, field); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}

	wantInlineMessage := &api.Message{
		ID:            ".package.Message.arrayWithObject",
		Name:          "arrayWithObject",
		Documentation: "The message type for the [arrayWithObject][package.Message.arrayWithObject] field.",
		Fields: []*api.Field{
			{
				Name:          "stringField",
				JSONName:      "stringField",
				ID:            ".package.Message.arrayWithObject.stringField",
				Documentation: "The stringField field.",
				Typez:         api.STRING_TYPE,
				TypezID:       "string",
			},
			{
				Name:          "intField",
				JSONName:      "intField",
				ID:            ".package.Message.arrayWithObject.intField",
				Documentation: "The intField field.",
				Typez:         api.UINT64_TYPE,
				TypezID:       "uint64",
			},
		},
		Parent: message,
	}
	gotInlineMessage, ok := model.State.MessageByID[wantInlineMessage.ID]
	if !ok {
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
	message := &api.Message{ID: ".package.Message"}
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
	message := &api.Message{ID: ".package.Message"}
	if field, err := makeArrayField(model, message, input); err == nil {
		t.Errorf("expected an error with an invalid inline object, got=%v", field)
	}
}
