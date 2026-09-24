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
	m0 := NewTestMessage("Message0").WithDocumentation(Input)
	model := NewTestAPI([]*Message{m0}, nil, nil)
	overrides := []DocumentationOverride{
		{
			ID:      ".test.Message0",
			Match:   Match,
			Replace: Replace,
		},
	}
	if err := PatchDocumentation(model, overrides); err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(m0.Documentation, Want); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func testPatchCommentsModel() *API {
	m0 := NewTestMessage("Message0").
		WithDocumentation(Input).
		WithFields(NewTestField("Field0").WithDocumentation(Input))
	e0 := NewTestEnum("Enum0").
		WithDocumentation(Input).
		WithValues(NewTestEnumValue("EV0", 0).WithDocumentation(Input))
	s0 := NewTestService("Service0").
		WithDocumentation(Input).
		WithMethods(NewTestMethod("Method0").WithDocumentation(Input))
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
		overrides := []DocumentationOverride{
			{
				ID:      id,
				Match:   Match,
				Replace: Replace,
			},
		}
		if err := PatchDocumentation(model, overrides); err == nil {
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
		overrides := []DocumentationOverride{
			{
				ID:      id,
				Match:   "NOT A STRING WE WILL FIND",
				Replace: Replace,
			},
		}
		if err := PatchDocumentation(model, overrides); err == nil {
			t.Errorf("expected an error replacing comments for entity %q", id)
		}
	}
}

func TestPatchCommentsField(t *testing.T) {
	f0 := NewTestField("field_name").WithDocumentation(Input)
	m0 := NewTestMessage("Message0").WithFields(f0)
	model := NewTestAPI([]*Message{m0}, nil, nil)
	overrides := []DocumentationOverride{
		{
			ID:      ".test.Message0.field_name",
			Match:   Match,
			Replace: Replace,
		},
	}
	if err := PatchDocumentation(model, overrides); err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(f0.Documentation, Want); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestPatchCommentsEnum(t *testing.T) {
	e0 := NewTestEnum("Enum0").WithDocumentation(Input)
	model := NewTestAPI(nil, []*Enum{e0}, nil)
	overrides := []DocumentationOverride{
		{
			ID:      ".test.Enum0",
			Match:   Match,
			Replace: Replace,
		},
	}
	if err := PatchDocumentation(model, overrides); err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(e0.Documentation, Want); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestPatchCommentsEnumValue(t *testing.T) {
	v0 := NewTestEnumValue("ENUM_VALUE", 0).WithDocumentation(Input)
	e0 := NewTestEnum("Enum0").WithDocumentation(Input).WithValues(v0)
	model := NewTestAPI(nil, []*Enum{e0}, nil)
	overrides := []DocumentationOverride{
		{
			ID:      ".test.Enum0.ENUM_VALUE",
			Match:   Match,
			Replace: Replace,
		},
	}
	if err := PatchDocumentation(model, overrides); err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(v0.Documentation, Want); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestPatchCommentsService(t *testing.T) {
	s0 := NewTestService("Service0").WithDocumentation(Input)
	model := NewTestAPI(nil, nil, []*Service{s0})
	overrides := []DocumentationOverride{
		{
			ID:      ".test.Service0",
			Match:   Match,
			Replace: Replace,
		},
	}
	if err := PatchDocumentation(model, overrides); err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(s0.Documentation, Want); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestPatchCommentsMethod(t *testing.T) {
	m0 := NewTestMethod("Method").WithDocumentation(Input)
	s0 := NewTestService("Service0").WithMethods(m0)
	model := NewTestAPI(nil, nil, []*Service{s0})
	overrides := []DocumentationOverride{
		{
			ID:      ".test.Service0.Method",
			Match:   Match,
			Replace: Replace,
		},
	}
	if err := PatchDocumentation(model, overrides); err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(m0.Documentation, Want); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
