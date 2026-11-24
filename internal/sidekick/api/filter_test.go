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

package api

import (
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestAPI_Filter_Nil(t *testing.T) {
	if (*API)(nil).Filter(func(string) bool { return true }) != nil {
		t.Errorf("(*API)(nil).Filter() should return nil")
	}

	m1 := NewTestMessage("M1")
	e1 := NewTestEnum("E1")
	extM1 := NewTestMessage("ExtM1").WithPackage("google.type")
	extE1 := NewTestEnum("ExtE1").WithPackage("google.type")

	model := NewTestAPI([]*Message{m1}, []*Enum{e1}, nil)
	model.AddMessage(extM1)
	model.AddEnum(extE1)

	got := model.Filter(nil)
	if diff := cmp.Diff([]*Message{m1}, got.Messages, allowUnexported()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]*Enum{e1}, got.Enums, allowUnexported()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if !slices.Contains(got.ExternalMessages, extM1) || !slices.Contains(got.ExternalMessages, model.Message(".google.protobuf.Any")) {
		t.Errorf("ExternalMessages = %v, want extM1 and pre-registered external messages retained", got.ExternalMessages)
	}
	if !slices.Contains(got.ExternalEnums, extE1) || !slices.Contains(got.ExternalEnums, model.Enum(".google.protobuf.NullValue")) {
		t.Errorf("ExternalEnums = %v, want extE1 and pre-registered external enums retained", got.ExternalEnums)
	}
}

func TestAPI_Filter(t *testing.T) {
	m1 := NewTestMessage("M1")
	m2 := NewTestMessage("M2")
	e1 := NewTestEnum("E1")
	e2 := NewTestEnum("E2")
	extM1 := NewTestMessage("ExtM1").WithPackage("google.type")
	extM2 := NewTestMessage("ExtM2").WithPackage("google.type")
	extE1 := NewTestEnum("ExtE1").WithPackage("google.type")
	extE2 := NewTestEnum("ExtE2").WithPackage("google.type")
	meth1 := NewTestMethod("Method1").WithInput(m1).WithOutput(m1)
	s1 := NewTestService("S1").WithMethods(meth1)
	s2 := NewTestService("S2")
	res := NewTestResource("test.googleapis.com/Book")
	msgRes := NewTestResource("test.googleapis.com/Shelf")
	mapType := NewTestMap("AttributesEntry", TypezString, TypezString)

	model := NewTestAPI([]*Message{m1, m2}, []*Enum{e1, e2}, []*Service{s1, s2})
	model.Name = "testapi"
	model.Title = "Test API"
	model.Description = "Test API description"
	model.Revision = "2026-01-01"
	model.RubyPackage = "Test::Api"
	model.PhpNamespace = "Test\\Api"
	model.CsharpNamespace = "Test.Api"
	model.QuickstartService = s1
	model.Codec = "orig-codec"
	model.ResourceDefinitions = []*Resource{res}
	s1.SetAnnotations("svc-codec")
	model.AddMessage(extM1)
	model.AddMessage(extM2)
	model.AddEnum(extE1)
	model.AddEnum(extE2)
	model.AddMap(mapType)
	model.AddResource(res)
	model.AddResource(msgRes)

	filtered := model.Filter(func(id string) bool {
		return id == m1.ID() || id == e1.ID() || id == extM1.ID() || id == extE1.ID()
	})

	if len(filtered.Messages) != 1 || filtered.Messages[0] != m1 {
		t.Errorf("got Messages = %v, want [%v]", filtered.Messages, m1)
	}
	if len(filtered.Enums) != 1 || filtered.Enums[0] != e1 {
		t.Errorf("got Enums = %v, want [%v]", filtered.Enums, e1)
	}
	if len(filtered.ExternalMessages) != 1 || filtered.ExternalMessages[0] != extM1 {
		t.Errorf("got ExternalMessages = %v, want [%v]", filtered.ExternalMessages, extM1)
	}
	if len(filtered.ExternalEnums) != 1 || filtered.ExternalEnums[0] != extE1 {
		t.Errorf("got ExternalEnums = %v, want [%v]", filtered.ExternalEnums, extE1)
	}
	if got := filtered.Message(m2.ID()); got != m2 {
		t.Errorf("filtered.Message(%q) = %v, want %v", m2.ID(), got, m2)
	}
	if got := filtered.Enum(e2.ID()); got != e2 {
		t.Errorf("filtered.Enum(%q) = %v, want %v", e2.ID(), got, e2)
	}
	if got := filtered.Map(mapType.ID()); got != mapType {
		t.Errorf("filtered.Map(%q) = %v, want %v", mapType.ID(), got, mapType)
	}
	if got := filtered.Resource(res.Type()); got != res {
		t.Errorf("filtered.Resource(%q) = %v, want %v", res.Type(), got, res)
	}
	if got := filtered.Resource(msgRes.Type()); got != msgRes {
		t.Errorf("filtered.Resource(%q) = %v, want %v", msgRes.Type(), got, msgRes)
	}
	if filtered.Codec != nil {
		t.Errorf("filtered.Codec = %v, want nil", filtered.Codec)
	}

	services := filtered.Services
	if len(services) != 2 {
		t.Fatalf("len(filtered.Services) = %d, want 2", len(services))
	}
	if services[0] == s1 || services[0].Codec() != nil {
		t.Errorf("cloned service did not preserve Clone() semantics")
	}
	if filtered.QuickstartService != services[0] {
		t.Errorf("filtered.QuickstartService = %v, want %v", filtered.QuickstartService, services[0])
	}
	if services[0].Model() != filtered {
		t.Errorf("services[0].Model() = %v, want %v", services[0].Model(), filtered)
	}
	methods := services[0].Methods()
	if len(methods) != 1 || methods[0].Model() != filtered {
		t.Errorf("cloned method Model() = %v, want %v", methods[0].Model(), filtered)
	}
	if filtered.Name != model.Name ||
		filtered.PackageName != model.PackageName ||
		filtered.Title != model.Title ||
		filtered.Description != model.Description ||
		filtered.Revision != model.Revision ||
		filtered.RubyPackage != model.RubyPackage ||
		filtered.PhpNamespace != model.PhpNamespace ||
		filtered.CsharpNamespace != model.CsharpNamespace {
		t.Errorf("API metadata was not preserved: got %+v", filtered)
	}
}
