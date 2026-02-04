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

package api

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/config"
)

const (
	Input = `Title

Thing to preserve

Valid versions are:
  Article Suggestion baseline model:
    - 0.9
    - 1.0 (default)
  Summarization baseline model:
    - 1.0

More things that are preserved.
`

	Want = `Title

Thing to preserve

Valid versions are:
* Article Suggestion baseline model:
    - 0.9
    - 1.0 (default)
* Summarization baseline model:
    - 1.0

More things that are preserved.
`
	Match = `Valid versions are:
  Article Suggestion baseline model:
    - 0.9
    - 1.0 (default)
  Summarization baseline model:
    - 1.0`
	Replace = `Valid versions are:
* Article Suggestion baseline model:
    - 0.9
    - 1.0 (default)
* Summarization baseline model:
    - 1.0`
)

func TestPatchCommentsMessage(t *testing.T) {
	m0 := &Message{
		Name:          "Message0",
		Package:       "test",
		ID:            ".test.Message0",
		Documentation: Input,
	}
	model := NewTestAPI([]*Message{m0}, []*Enum{}, []*Service{})
	cfg := config.Config{
		CommentOverrides: []config.DocumentationOverride{
			{
				ID:      ".test.Message0",
				Match:   Match,
				Replace: Replace,
			},
		},
	}
	if err := PatchDocumentation(model, cfg.CommentOverrides); err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(m0.Documentation, Want); diff != "" {
		t.Errorf("mismatch in enums (-want, +got)\n:%s", diff)
	}
}

func testPatchCommentsModel() *API {
	m0 := &Message{
		Name:          "Message0",
		Package:       "test",
		ID:            ".test.Message0",
		Documentation: Input,
		Fields: []*Field{
			{
				Name:          "Field0",
				ID:            ".test.Message0.Field0",
				Documentation: Input,
			},
		},
	}
	e0 := &Enum{
		Name:          "Enum0",
		Package:       "test",
		ID:            ".test.Enum0",
		Documentation: Input,
		Values: []*EnumValue{
			{
				Name:          "EV0",
				ID:            ".test.Enum0.EV0",
				Documentation: Input,
			},
		},
	}
	s0 := &Service{
		Name:          "Service0",
		Package:       "test",
		ID:            ".test.Service0",
		Documentation: Input,
		Methods: []*Method{
			{
				Name:          "Method0",
				ID:            ".test.Service0.Method0",
				Documentation: Input,
			},
		},
	}
	return NewTestAPI([]*Message{m0}, []*Enum{e0}, []*Service{s0})
}

func TestPatchCommentsMessageNotFound(t *testing.T) {
	model := testPatchCommentsModel()

	missing := []string{
		".test.MissingMessage",
		".test.Message0.MissingField",
		".test.Enum0.MissingEnumValue",
		".test.Service0.MissingMethod",
		"NotAThing",
	}
	for _, id := range missing {
		cfg := config.Config{
			CommentOverrides: []config.DocumentationOverride{
				{
					ID:      id,
					Match:   Match,
					Replace: Replace,
				},
			},
		}
		if err := PatchDocumentation(model, cfg.CommentOverrides); err == nil {
			t.Errorf("expected an error searching for missing entity %q", id)
		}
	}
}

func TestPatchCommentsNoMatch(t *testing.T) {
	model := testPatchCommentsModel()

	missing := []string{
		".test.Message0",
		".test.Message0.Field0",
		".test.Enum0",
		".test.Enum0.EV0",
		".test.Service0",
		".test.Service0.Method0",
	}
	for _, id := range missing {
		cfg := config.Config{
			CommentOverrides: []config.DocumentationOverride{
				{
					ID:      id,
					Match:   "NOT A STRING WE WILL FIND",
					Replace: Replace,
				},
			},
		}
		if err := PatchDocumentation(model, cfg.CommentOverrides); err == nil {
			t.Errorf("expected an error replacing comments for entity %q", id)
		}
	}
}

func TestPatchCommentsField(t *testing.T) {
	f0 := &Field{
		Name:          "field_name",
		ID:            ".test.Message0.field_name",
		Documentation: Input,
	}
	m0 := &Message{
		Name:    "Message0",
		Package: "test",
		ID:      ".test.Message0",
		Fields:  []*Field{f0},
	}
	model := NewTestAPI([]*Message{m0}, []*Enum{}, []*Service{})
	cfg := config.Config{
		CommentOverrides: []config.DocumentationOverride{
			{
				ID:      ".test.Message0.field_name",
				Match:   Match,
				Replace: Replace,
			},
		},
	}
	if err := PatchDocumentation(model, cfg.CommentOverrides); err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(f0.Documentation, Want); diff != "" {
		t.Errorf("mismatch in enums (-want, +got)\n:%s", diff)
	}
}

func TestPatchCommentsEnum(t *testing.T) {
	e0 := &Enum{
		Name:          "Enum0",
		Package:       "test",
		ID:            ".test.Enum0",
		Documentation: Input,
	}
	model := NewTestAPI([]*Message{}, []*Enum{e0}, []*Service{})
	cfg := config.Config{
		CommentOverrides: []config.DocumentationOverride{
			{
				ID:      ".test.Enum0",
				Match:   Match,
				Replace: Replace,
			},
		},
	}
	if err := PatchDocumentation(model, cfg.CommentOverrides); err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(e0.Documentation, Want); diff != "" {
		t.Errorf("mismatch in enums (-want, +got)\n:%s", diff)
	}
}

func TestPatchCommentsEnumValue(t *testing.T) {
	v0 := &EnumValue{
		Name:          "ENUM_VALUE",
		ID:            ".test.Enum0.ENUM_VALUE",
		Documentation: Input,
	}
	e0 := &Enum{
		Name:          "Enum0",
		Package:       "test",
		ID:            ".test.Enum0",
		Values:        []*EnumValue{v0},
		Documentation: Input,
	}
	model := NewTestAPI([]*Message{}, []*Enum{e0}, []*Service{})
	cfg := config.Config{
		CommentOverrides: []config.DocumentationOverride{
			{
				ID:      ".test.Enum0.ENUM_VALUE",
				Match:   Match,
				Replace: Replace,
			},
		},
	}
	if err := PatchDocumentation(model, cfg.CommentOverrides); err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(v0.Documentation, Want); diff != "" {
		t.Errorf("mismatch in enums (-want, +got)\n:%s", diff)
	}
}

func TestPatchCommentsService(t *testing.T) {
	s0 := &Service{
		Name:          "Service0",
		Package:       "test",
		ID:            ".test.Service0",
		Documentation: Input,
	}
	model := NewTestAPI([]*Message{}, []*Enum{}, []*Service{s0})
	cfg := config.Config{
		CommentOverrides: []config.DocumentationOverride{
			{
				ID:      ".test.Service0",
				Match:   Match,
				Replace: Replace,
			},
		},
	}
	if err := PatchDocumentation(model, cfg.CommentOverrides); err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(s0.Documentation, Want); diff != "" {
		t.Errorf("mismatch in enums (-want, +got)\n:%s", diff)
	}
}

func TestPatchCommentsMethod(t *testing.T) {
	m0 := &Method{
		Name:          "Method",
		ID:            ".test.Service0.Method",
		Documentation: Input,
	}
	s0 := &Service{
		Name:    "Service0",
		Package: "test",
		ID:      ".test.Service0",
		Methods: []*Method{m0},
	}
	model := NewTestAPI([]*Message{}, []*Enum{}, []*Service{s0})
	cfg := config.Config{
		CommentOverrides: []config.DocumentationOverride{
			{
				ID:      ".test.Service0.Method",
				Match:   Match,
				Replace: Replace,
			},
		},
	}
	if err := PatchDocumentation(model, cfg.CommentOverrides); err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(m0.Documentation, Want); diff != "" {
		t.Errorf("mismatch in enums (-want, +got)\n:%s", diff)
	}
}
