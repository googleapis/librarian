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

func TestDeprecatedAPI(t *testing.T) {
	newAPI := func() *API {
		enums := []*Enum{{Name: "e1", Package: "p1"}}
		messages := []*Message{{Name: "m1", Package: "p1"}}
		services := []*Service{{Name: "s1"}}
		return NewTestAPI(messages, enums, services)
	}

	model := newAPI()
	if model.HasDeprecatedEntities() {
		t.Errorf("expected no deprecated in baseline %v", model)
	}

	model = newAPI()
	model.Enums[0].Deprecated = true
	if !model.HasDeprecatedEntities() {
		t.Errorf("deprecated enum should result in deprecated entities for model %v", model)
	}

	model = newAPI()
	model.Messages[0].Deprecated = true
	if !model.HasDeprecatedEntities() {
		t.Errorf("deprecated message should result in deprecated entities for model %v", model)
	}

	model = newAPI()
	model.Services[0].Deprecated = true
	if !model.HasDeprecatedEntities() {
		t.Errorf("deprecated service should result in deprecated entities for model %v", model)
	}
}

func TestDeprecatedMessage(t *testing.T) {
	m1 := &Message{
		Name:       "m1",
		Package:    "p1",
		Deprecated: true,
		Fields: []*Field{
			{Name: "f1"},
			{Name: "f2"},
		},
	}
	if !m1.hasDeprecatedEntities() {
		t.Errorf("expected deprecated entities in message %v", m1)
	}

	m2 := &Message{
		Name:    "m2",
		Package: "p1",
		Fields: []*Field{
			{Name: "f1", Deprecated: true},
			{Name: "f2"},
		},
	}
	if !m2.hasDeprecatedEntities() {
		t.Errorf("expected deprecated entities in message %v", m2)
	}

	m3 := &Message{
		Name:    "m3",
		Package: "p1",
		Messages: []*Message{
			{Name: "child1", Deprecated: true},
		},
		Fields: []*Field{
			{Name: "f1"},
			{Name: "f2"},
		},
	}
	if !m3.hasDeprecatedEntities() {
		t.Errorf("expected deprecated entities in message %v", m3)
	}

	m4 := &Message{
		Name:    "m4",
		Package: "p1",
		Messages: []*Message{
			{Name: "child1"},
		},
		Enums: []*Enum{
			{Name: "enum1", Deprecated: true},
		},
		Fields: []*Field{
			{Name: "f1"},
			{Name: "f2"},
		},
	}
	if !m4.hasDeprecatedEntities() {
		t.Errorf("expected deprecated entities in message %v", m4)
	}

	m5 := &Message{
		Name:    "m5",
		Package: "p1",
		Messages: []*Message{
			{Name: "child1"},
		},
		Enums: []*Enum{
			{Name: "enum1"},
		},
		Fields: []*Field{
			{Name: "f1"},
			{Name: "f2"},
		},
	}
	if m5.hasDeprecatedEntities() {
		t.Errorf("expected no deprecated entities in message %v", m5)
	}
}

func TestDeprecatedEnum(t *testing.T) {
	e1 := &Enum{
		Name:       "e1",
		Package:    "p1",
		Deprecated: true,
		Values: []*EnumValue{
			{Name: "V1", Number: 1},
			{Name: "V2", Number: 2},
		},
	}
	if !e1.hasDeprecatedEntities() {
		t.Errorf("expected deprecated entities in enum %v", e1)
	}

	e2 := &Enum{
		Name:    "e2",
		Package: "p1",
		Values: []*EnumValue{
			{Name: "V1", Number: 1},
			{Name: "V2", Number: 2, Deprecated: true},
		},
	}
	if !e2.hasDeprecatedEntities() {
		t.Errorf("expected deprecated entities in enum %v", e2)
	}

	e3 := &Enum{
		Name:    "e3",
		Package: "p1",
		Values: []*EnumValue{
			{Name: "V1", Number: 1},
			{Name: "V2", Number: 2},
		},
	}
	if e3.hasDeprecatedEntities() {
		t.Errorf("expected no deprecated entities in enum %v", e3)
	}
}

func TestDeprecatedService(t *testing.T) {
	s1 := &Service{
		Name:       "s1",
		Package:    "p1",
		Deprecated: true,
		Methods: []*Method{
			{Name: "m1"},
			{Name: "m2"},
		},
	}
	if !s1.hasDeprecatedEntities() {
		t.Errorf("expected deprecated entities in enum %v", s1)
	}

	s2 := &Service{
		Name:    "s2",
		Package: "p1",
		Methods: []*Method{
			{Name: "m1", Deprecated: true},
			{Name: "m2"},
		},
	}
	if !s2.hasDeprecatedEntities() {
		t.Errorf("expected deprecated entities in enum %v", s2)
	}

	s3 := &Service{
		Name:    "s3",
		Package: "p1",
		Methods: []*Method{
			{Name: "m1"},
			{Name: "m2"},
		},
	}
	if s3.hasDeprecatedEntities() {
		t.Errorf("expected no deprecated entities in enum %v", s3)
	}
}
