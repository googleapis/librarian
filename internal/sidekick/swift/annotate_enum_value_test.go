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

	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateEnumValue(t *testing.T) {
	enum := &api.Enum{Name: "Color"}
	ev := &api.EnumValue{Name: "COLOR_RED", Number: 1, Documentation: "Red color", Parent: enum}

	codec := &codec{}
	codec.annotateEnumValue(ev)

	ann, ok := ev.Codec.(*enumValueAnnotations)
	if !ok {
		t.Fatal("expected enumValueAnnotations")
	}

	if ann.Name != "red" {
		t.Errorf("ann.Name = %q, want %q", ann.Name, "red")
	}
	if ann.Number != 1 {
		t.Errorf("ann.Number = %d, want %d", ann.Number, 1)
	}
	if ann.StringValue != "COLOR_RED" {
		t.Errorf("ann.StringValue = %q, want %q", ann.StringValue, "COLOR_RED")
	}
}

func TestAnnotateEnum_Duplicates(t *testing.T) {
	enum := &api.Enum{
		Name: "Color",
	}
	enum.Values = []*api.EnumValue{
		{Name: "COLOR_RED", Number: 1, Parent: enum},
		{Name: "RED", Number: 2, Parent: enum},
	}

	codec := &codec{}
	codec.annotateEnum(enum, &modelAnnotations{})

	ann, ok := enum.Codec.(*enumAnnotations)
	if !ok {
		t.Fatal("expected enumAnnotations")
	}

	if len(ann.Values) != 1 {
		t.Errorf("len(ann.Values) = %d, want 1", len(ann.Values))
	}

	if ann.Values[0].Name != "red" {
		t.Errorf("ann.Values[0].Name = %q, want %q", ann.Values[0].Name, "red")
	}
}
