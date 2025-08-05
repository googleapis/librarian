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

func TestScopesService(t *testing.T) {
	service := &Service{
		Name:    "Service",
		Package: "test",
		ID:      ".test.Service",
	}
	got := service.Scopes()
	want := []string{"test.Service", "test"}
	if diff := cmp.Diff(want, got); len(diff) != 0 {
		t.Errorf("mismatched service.Scopes() (-want, +got):\n%s", diff)
	}
}

func TestScopesMessage(t *testing.T) {
	parent := &Message{
		Name:    "Parent",
		Package: "test",
		ID:      ".test.Parent",
	}
	child := &Message{
		Name:    "Child",
		Package: "test",
		ID:      ".test.Parent.Child",
		Parent:  parent,
	}
	parent.Messages = []*Message{child}

	got := parent.Scopes()
	want := []string{"test.Parent", "test"}
	if diff := cmp.Diff(want, got); len(diff) != 0 {
		t.Errorf("mismatched parent.Scopes() (-want, +got):\n%s", diff)
	}

	got = child.Scopes()
	want = []string{"test.Parent.Child", "test.Parent", "test"}
	if diff := cmp.Diff(want, got); len(diff) != 0 {
		t.Errorf("mismatched child.Scopes() (-want, +got):\n%s", diff)
	}
}

func TestScopesEnum(t *testing.T) {
	enum := &Enum{
		Name:    "Enum",
		Package: "test",
		ID:      ".test.Enum",
	}

	got := enum.Scopes()
	want := []string{"test.Enum", "test"}
	if diff := cmp.Diff(want, got); len(diff) != 0 {
		t.Errorf("mismatched enum.Scopes() (-want, +got):\n%s", diff)
	}
}

func TestScopesEnumInMessage(t *testing.T) {
	parent := &Message{
		Name:    "Parent",
		Package: "test",
		ID:      ".test.Parent",
	}
	child := &Enum{
		Name:    "Child",
		Package: "test",
		ID:      ".test.Parent.Child",
		Parent:  parent,
	}
	parent.Enums = []*Enum{child}

	got := child.Scopes()
	want := []string{"test.Parent.Child", "test.Parent", "test"}
	if diff := cmp.Diff(want, got); len(diff) != 0 {
		t.Errorf("mismatched child.Scopes() (-want, +got):\n%s", diff)
	}
}

func TestScopesEnumValue(t *testing.T) {
	enum := &Enum{
		Name:    "Enum",
		Package: "test",
		ID:      ".test.Enum",
	}
	enumValue := &EnumValue{
		Name:   "EV",
		ID:     ".test.Enum.EV",
		Parent: enum,
	}
	enum.Values = []*EnumValue{enumValue}

	got := enumValue.Scopes()
	want := []string{"test.Enum", "test"}
	if diff := cmp.Diff(want, got); len(diff) != 0 {
		t.Errorf("mismatched enum.Scopes() (-want, +got):\n%s", diff)
	}
}

func TestScopesEnumValueInMessage(t *testing.T) {
	parent := &Message{
		Name:    "Parent",
		Package: "test",
		ID:      ".test.Parent",
	}
	enum := &Enum{
		Name:    "Enum",
		Package: "test",
		ID:      ".test.Parent.Enum",
		Parent:  parent,
	}
	enumValue := &EnumValue{
		Name:   "EV",
		ID:     ".test.Parent.Enum.EV",
		Parent: enum,
	}
	enum.Values = []*EnumValue{enumValue}
	parent.Enums = []*Enum{enum}

	got := enumValue.Scopes()
	want := []string{"test.Parent.Enum", "test.Parent", "test"}
	if diff := cmp.Diff(want, got); len(diff) != 0 {
		t.Errorf("mismatched child.Scopes() (-want, +got):\n%s", diff)
	}
}
