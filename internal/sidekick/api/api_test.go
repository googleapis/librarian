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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestAPI(t *testing.T) {
	msg1 := NewMessageBuilder().WithName("Msg1").WithID(".test.Msg1").Build()
	msg2 := NewMessageBuilder().WithName("Msg2").WithID(".test.Msg2").Build()
	extMsg := NewMessageBuilder().WithName("ExtMsg").WithID(".ext.Msg").Build()
	en := NewEnumBuilder().WithName("En").WithID(".test.En").Build()
	extEn := NewEnumBuilder().WithName("ExtEn").WithID(".ext.En").Build()
	svc := NewServiceBuilder().WithName("Svc").WithID(".test.Svc").Build()
	r1 := NewResourceBuilder().WithType("test.googleapis.com/R1").Build()
	r2 := NewResourceBuilder().WithType("test.googleapis.com/R2").Build()

	got := NewAPIBuilder().
		WithName("test").
		WithPackageName("testpkg").
		WithTitle("Test API").
		WithDescription("Test description.").
		WithRevision("v1").
		WithRubyPackage("Test::V1").
		WithPhpNamespace("Test\\V1").
		WithCsharpNamespace("Test.V1").
		WithMessages(msg1).
		AppendMessage(msg2).
		WithExternalMessages(extMsg).
		WithEnums(en).
		WithExternalEnums(extEn).
		WithServices(svc).
		WithResourceDefinitions(r1).
		AppendResourceDefinitions(r2).
		WithQuickstartService(svc).
		AddMessage(msg1).
		AddEnum(en).
		AddService(svc).
		AddResource(r1).
		BuildSyntaxTree()

	got.SetAnnotations("api-codec")

	want := &API{
		name:                "test",
		packageName:         "testpkg",
		title:               "Test API",
		description:         "Test description.",
		revision:            "v1",
		rubyPackage:         "Test::V1",
		phpNamespace:        "Test\\V1",
		csharpNamespace:     "Test.V1",
		messages:            []*Message{msg1, msg2},
		externalMessages:    []*Message{extMsg},
		enums:               []*Enum{en},
		externalEnums:       []*Enum{extEn},
		services:            []*Service{svc},
		resourceDefinitions: []*Resource{r1, r2},
		quickstartService:   svc,
		messageByID:         map[string]*Message{".test.Msg1": msg1},
		enumByID:            map[string]*Enum{".test.En": en},
		serviceByID:         map[string]*Service{".test.Svc": svc},
		methodByID:          map[string]*Method{},
		resourceByType:      map[string]*Resource{"test.googleapis.com/R1": r1},
		mapByID:             map[string]*Map{},
		codec:               "api-codec",
	}
	if diff := cmp.Diff(want, got, allowUnexported()); diff != "" {
		t.Errorf("API mismatch (-want +got):\n%s", diff)
	}
	if got.Name() != want.name || got.PackageName() != want.packageName || got.Title() != want.title || got.Description() != want.description || got.Revision() != want.revision || got.RubyPackage() != want.rubyPackage || got.PhpNamespace() != want.phpNamespace || got.CsharpNamespace() != want.csharpNamespace || got.QuickstartService() != svc || got.Codec() != "api-codec" {
		t.Errorf("unexpected API fields: %+v", got)
	}
	if len(got.Messages()) != 2 || got.Messages()[0] != msg1 || got.Messages()[1] != msg2 || len(got.ExternalMessages()) != 1 || got.ExternalMessages()[0] != extMsg || len(got.Enums()) != 1 || got.Enums()[0] != en || len(got.ExternalEnums()) != 1 || got.ExternalEnums()[0] != extEn || len(got.Services()) != 1 || got.Services()[0] != svc || len(got.ResourceDefinitions()) != 2 || got.ResourceDefinitions()[0] != r1 || got.ResourceDefinitions()[1] != r2 {
		t.Errorf("API slice inputs were not cloned")
	}

	got.Messages()[0] = nil
	got.ExternalMessages()[0] = nil
	got.Enums()[0] = nil
	got.ExternalEnums()[0] = nil
	got.Services()[0] = nil
	got.ResourceDefinitions()[0] = nil
	if got.Messages()[0] != msg1 || got.ExternalMessages()[0] != extMsg || got.Enums()[0] != en || got.ExternalEnums()[0] != extEn || got.Services()[0] != svc || got.ResourceDefinitions()[0] != r1 {
		t.Errorf("API slice getters should return defensive copies")
	}

	a2 := NewAPIBuilderFrom(got).WithName("test2").BuildSyntaxTree()
	if a2.Name() != "test2" || len(a2.Messages()) != 2 || a2.Message(".test.Msg1") != msg1 {
		t.Errorf("NewAPIBuilderFrom should copy maps and slices: %+v", a2)
	}
}
