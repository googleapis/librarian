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
		enums := []*Enum{NewTestEnum("e1").WithPackage("p1")}
		messages := []*Message{NewTestMessage("m1").WithPackage("p1")}
		services := []*Service{NewTestService("s1")}
		return NewTestAPI(messages, enums, services)
	}

	model := newAPI()
	if model.HasDeprecatedEntities() {
		t.Errorf("expected no deprecated in baseline %v", model)
	}

	model = newAPI()
	model.Enums[0].WithDeprecated(true)
	if !model.HasDeprecatedEntities() {
		t.Errorf("deprecated enum should result in deprecated entities for model %v", model)
	}

	model = newAPI()
	model.Messages[0].WithDeprecated(true)
	if !model.HasDeprecatedEntities() {
		t.Errorf("deprecated message should result in deprecated entities for model %v", model)
	}

	model = newAPI()
	model.Services[0].WithDeprecated(true)
	if !model.HasDeprecatedEntities() {
		t.Errorf("deprecated service should result in deprecated entities for model %v", model)
	}
}

func TestDeprecatedMessage(t *testing.T) {
	m1 := NewTestMessage("m1").
		WithPackage("p1").
		WithDeprecated(true).
		WithFields(NewTestField("f1"), NewTestField("f2"))
	if !m1.hasDeprecatedEntities() {
		t.Errorf("expected deprecated entities in message %v", m1)
	}

	m2 := NewTestMessage("m2").
		WithPackage("p1").
		WithFields(
			NewTestField("f1").WithDeprecated(true),
			NewTestField("f2"),
		)
	if !m2.hasDeprecatedEntities() {
		t.Errorf("expected deprecated entities in message %v", m2)
	}

	m3 := NewTestMessage("m3").
		WithPackage("p1").
		WithMessages(NewTestMessage("child1").WithDeprecated(true)).
		WithFields(NewTestField("f1"), NewTestField("f2"))
	if !m3.hasDeprecatedEntities() {
		t.Errorf("expected deprecated entities in message %v", m3)
	}

	m4 := NewTestMessage("m4").
		WithPackage("p1").
		WithMessages(NewTestMessage("child1")).
		WithEnums(NewTestEnum("enum1").WithDeprecated(true)).
		WithFields(NewTestField("f1"), NewTestField("f2"))
	if !m4.hasDeprecatedEntities() {
		t.Errorf("expected deprecated entities in message %v", m4)
	}

	m5 := NewTestMessage("m5").
		WithPackage("p1").
		WithMessages(NewTestMessage("child1")).
		WithEnums(NewTestEnum("enum1")).
		WithFields(NewTestField("f1"), NewTestField("f2"))
	if m5.hasDeprecatedEntities() {
		t.Errorf("expected no deprecated entities in message %v", m5)
	}
}

func TestDeprecatedEnum(t *testing.T) {
	e1 := NewTestEnum("e1").
		WithPackage("p1").
		WithDeprecated(true).
		WithValues(NewTestEnumValue("V1", 1), NewTestEnumValue("V2", 2))
	if !e1.hasDeprecatedEntities() {
		t.Errorf("expected deprecated entities in enum %v", e1)
	}

	e2 := NewTestEnum("e2").
		WithPackage("p1").
		WithValues(
			NewTestEnumValue("V1", 1),
			NewTestEnumValue("V2", 2).WithDeprecated(true),
		)
	if !e2.hasDeprecatedEntities() {
		t.Errorf("expected deprecated entities in enum %v", e2)
	}

	e3 := NewTestEnum("e3").
		WithPackage("p1").
		WithValues(NewTestEnumValue("V1", 1), NewTestEnumValue("V2", 2))
	if e3.hasDeprecatedEntities() {
		t.Errorf("expected no deprecated entities in enum %v", e3)
	}
}

func TestDeprecatedService(t *testing.T) {
	s1 := NewTestService("s1").
		WithPackage("p1").
		WithDeprecated(true).
		WithMethods(NewTestMethod("m1"), NewTestMethod("m2"))
	if !s1.hasDeprecatedEntities() {
		t.Errorf("expected deprecated entities in enum %v", s1)
	}

	s2 := NewTestService("s2").
		WithPackage("p1").
		WithMethods(
			NewTestMethod("m1").WithDeprecated(true),
			NewTestMethod("m2"),
		)
	if !s2.hasDeprecatedEntities() {
		t.Errorf("expected deprecated entities in enum %v", s2)
	}

	s3 := NewTestService("s3").
		WithPackage("p1").
		WithMethods(NewTestMethod("m1"), NewTestMethod("m2"))
	if s3.hasDeprecatedEntities() {
		t.Errorf("expected no deprecated entities in enum %v", s3)
	}
}
