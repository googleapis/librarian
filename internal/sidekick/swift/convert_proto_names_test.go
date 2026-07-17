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

func TestProtoMessageAndEnumTypeName(t *testing.T) {
	parentMsg := &api.Message{
		Name:    "OuterMessage",
		ID:      ".test.OuterMessage",
		Package: "test",
	}
	nestedMsg := &api.Message{
		Name:    "InnerMessage",
		ID:      ".test.OuterMessage.InnerMessage",
		Package: "test",
		Parent:  parentMsg,
	}
	topEnum := &api.Enum{
		Name:    "TopEnum",
		ID:      ".test.TopEnum",
		Package: "test",
	}
	nestedEnum := &api.Enum{
		Name:    "NestedEnum",
		ID:      ".test.OuterMessage.NestedEnum",
		Package: "test",
		Parent:  parentMsg,
	}

	model := api.NewTestAPI([]*api.Message{parentMsg, nestedMsg}, []*api.Enum{topEnum, nestedEnum}, []*api.Service{})
	model.PackageName = "test"

	t.Run("with empty ModulePath", func(t *testing.T) {
		codec := newTestCodec(t, model, map[string]string{})
		
		gotMsg := codec.protoMessageTypeName(parentMsg)
		wantMsg := "Test_OuterMessage"
		if gotMsg != wantMsg {
			t.Errorf("protoMessageTypeName(parentMsg) = %q, want %q", gotMsg, wantMsg)
		}

		gotNestedMsg := codec.protoMessageTypeName(nestedMsg)
		wantNestedMsg := "Test_OuterMessage.InnerMessage"
		if gotNestedMsg != wantNestedMsg {
			t.Errorf("protoMessageTypeName(nestedMsg) = %q, want %q", gotNestedMsg, wantNestedMsg)
		}

		gotEnum := codec.protoEnumTypeName(topEnum)
		wantEnum := "Test_TopEnum"
		if gotEnum != wantEnum {
			t.Errorf("protoEnumTypeName(topEnum) = %q, want %q", gotEnum, wantEnum)
		}

		gotNestedEnum := codec.protoEnumTypeName(nestedEnum)
		wantNestedEnum := "Test_OuterMessage.NestedEnum"
		if gotNestedEnum != wantNestedEnum {
			t.Errorf("protoEnumTypeName(nestedEnum) = %q, want %q", gotNestedEnum, wantNestedEnum)
		}
	})

	t.Run("with populated ModulePath", func(t *testing.T) {
		codec := newTestCodec(t, model, map[string]string{
			"module-path": "TestProtos",
		})

		gotMsg := codec.protoMessageTypeName(parentMsg)
		wantMsg := "TestProtos.Test_OuterMessage"
		if gotMsg != wantMsg {
			t.Errorf("protoMessageTypeName(parentMsg) = %q, want %q", gotMsg, wantMsg)
		}

		gotNestedMsg := codec.protoMessageTypeName(nestedMsg)
		wantNestedMsg := "TestProtos.Test_OuterMessage.InnerMessage"
		if gotNestedMsg != wantNestedMsg {
			t.Errorf("protoMessageTypeName(nestedMsg) = %q, want %q", gotNestedMsg, wantNestedMsg)
		}

		gotEnum := codec.protoEnumTypeName(topEnum)
		wantEnum := "TestProtos.Test_TopEnum"
		if gotEnum != wantEnum {
			t.Errorf("protoEnumTypeName(topEnum) = %q, want %q", gotEnum, wantEnum)
		}

		gotNestedEnum := codec.protoEnumTypeName(nestedEnum)
		wantNestedEnum := "TestProtos.Test_OuterMessage.NestedEnum"
		if gotNestedEnum != wantNestedEnum {
			t.Errorf("protoEnumTypeName(nestedEnum) = %q, want %q", gotNestedEnum, wantNestedEnum)
		}
	})
}

func TestMessageAndEnumFileName(t *testing.T) {
	parentMsg := &api.Message{
		Name: "OuterMessage",
	}
	nestedMsg := &api.Message{
		Name:   "InnerMessage",
		Parent: parentMsg,
	}
	doubleNestedMsg := &api.Message{
		Name:   "LeafMessage",
		Parent: nestedMsg,
	}
	topEnum := &api.Enum{
		Name: "TopEnum",
	}
	nestedEnum := &api.Enum{
		Name:   "InnerEnum",
		Parent: parentMsg,
	}

	model := api.NewTestAPI([]*api.Message{parentMsg, nestedMsg, doubleNestedMsg}, []*api.Enum{topEnum, nestedEnum}, []*api.Service{})
	codec := newTestCodec(t, model, map[string]string{})

	t.Run("message conversion filenames", func(t *testing.T) {
		gotParent := codec.messageFileName(parentMsg)
		wantParent := "OuterMessage"
		if gotParent != wantParent {
			t.Errorf("messageFileName(parentMsg) = %q, want %q", gotParent, wantParent)
		}

		gotNested := codec.messageFileName(nestedMsg)
		wantNested := "OuterMessage+InnerMessage"
		if gotNested != wantNested {
			t.Errorf("messageFileName(nestedMsg) = %q, want %q", gotNested, wantNested)
		}

		gotLeaf := codec.messageFileName(doubleNestedMsg)
		wantLeaf := "OuterMessage+InnerMessage+LeafMessage"
		if gotLeaf != wantLeaf {
			t.Errorf("messageFileName(doubleNestedMsg) = %q, want %q", gotLeaf, wantLeaf)
		}
	})

	t.Run("enum conversion filenames", func(t *testing.T) {
		gotTop := codec.enumFileName(topEnum)
		wantTop := "TopEnum"
		if gotTop != wantTop {
			t.Errorf("enumFileName(topEnum) = %q, want %q", gotTop, wantTop)
		}

		gotNested := codec.enumFileName(nestedEnum)
		wantNested := "OuterMessage+InnerEnum"
		if gotNested != wantNested {
			t.Errorf("enumFileName(nestedEnum) = %q, want %q", gotNested, wantNested)
		}
	})
}
