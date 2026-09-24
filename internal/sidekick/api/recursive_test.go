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
)

func TestSimple(t *testing.T) {
	msg := NewTestMessage("Message")
	field0 := NewTestField("a").WithType(TypezString)
	field1 := NewTestField("b").WithMessageType(msg).WithOptional()
	msg.WithFields(field0, field1)
	model := NewTestAPI([]*Message{msg}, nil, nil)
	LabelRecursiveFields(model)
	if field0.Recursive {
		t.Errorf("mismatched IsRecursive field for %v", field0)
	}
	if !field1.Recursive {
		t.Errorf("mismatched IsRecursive field for %v", field1)
	}
}

func TestSimpleMap(t *testing.T) {
	parent := NewTestMessage("ParentMessage")
	key := NewTestField("key").WithType(TypezString)
	value := NewTestField("value").WithMessageType(parent)
	mapMessage := NewTestMapMessageWithFields("SingularMapEntry", key, value)
	parent.WithMessages(mapMessage)

	field0 := NewTestField("children").WithMessageType(mapMessage)
	parent.WithFields(field0)

	model := NewTestAPI([]*Message{parent, mapMessage}, nil, nil)
	LabelRecursiveFields(model)
	for _, field := range []*Field{value, field0} {
		if !field.Recursive {
			t.Errorf("expected IsRecursive to be true for field %s", field.ID)
		}
	}
	if key.Recursive {
		t.Errorf("expected IsRecursive to be false for field %s", key.ID)
	}
}

func TestIndirect(t *testing.T) {
	msg := NewTestMessage("Message")
	child := NewTestMessage("ChildMessage")
	grandChild := NewTestMessage("GrandChildMessage")

	field0 := NewTestField("child").WithMessageType(child).WithOptional()
	field1 := NewTestField("grand_child").WithMessageType(grandChild).WithOptional()
	field2 := NewTestField("back_to_grand_parent").WithMessageType(msg).WithOptional()

	msg.WithFields(field0)
	child.WithFields(field1)
	grandChild.WithFields(field2)

	model := NewTestAPI([]*Message{msg, child, grandChild}, nil, nil)
	LabelRecursiveFields(model)
	for _, field := range []*Field{field0, field1, field2} {
		if !field.Recursive {
			t.Errorf("IsRecursive should be true for field %s", field.Name)
		}
	}
}

func TestViaMap(t *testing.T) {
	parent := NewTestMessage("ParentMessage")
	child := NewTestMessage("ChildMessage")

	field0 := NewTestField("parent").WithMessageType(parent)
	child.WithFields(field0)

	key := NewTestField("key").WithType(TypezString)
	value := NewTestField("value").WithMessageType(child)
	mapMessage := NewTestMapMessageWithFields("SingularMapEntry", key, value)
	parent.WithMessages(mapMessage)

	field1 := NewTestField("children").WithMessageType(mapMessage)
	parent.WithFields(field1)

	model := NewTestAPI([]*Message{parent, child, mapMessage}, nil, nil)
	LabelRecursiveFields(model)
	for _, field := range []*Field{value, field0, field1} {
		if !field.Recursive {
			t.Errorf("expected IsRecursive to be true for field %s", field.ID)
		}
	}
	if key.Recursive {
		t.Errorf("expected IsRecursive to be false for field %s", key.ID)
	}
}

func TestReferencedCycle(t *testing.T) {
	parent := NewTestMessage("ParentMessage")
	child := NewTestMessage("ChildMessage")

	field0 := NewTestField("parent").WithMessageType(parent)
	child.WithFields(field0)

	field1 := NewTestField("child").WithMessageType(child)
	parent.WithFields(field1)

	field2 := NewTestField("ref").WithMessageType(parent)
	holder := NewTestMessage("Holder").WithFields(field2)

	model := NewTestAPI([]*Message{holder, parent, child}, nil, nil)
	LabelRecursiveFields(model)
	for _, field := range []*Field{field0, field1} {
		if !field.Recursive {
			t.Errorf("expected IsRecursive to be true for field %s", field.ID)
		}
	}
	for _, field := range []*Field{field2} {
		if field.Recursive {
			t.Errorf("expected IsRecursive to be false for field %s", field.ID)
		}
	}
}
