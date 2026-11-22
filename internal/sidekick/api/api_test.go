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

func TestAPIBuilder(t *testing.T) {
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

	got.Codec = "api-codec"

	want := &API{
		Name:                "test",
		PackageName:         "testpkg",
		Title:               "Test API",
		Description:         "Test description.",
		Revision:            "v1",
		RubyPackage:         "Test::V1",
		PhpNamespace:        "Test\\V1",
		CsharpNamespace:     "Test.V1",
		Messages:            []*Message{msg1, msg2},
		ExternalMessages:    []*Message{extMsg},
		Enums:               []*Enum{en},
		ExternalEnums:       []*Enum{extEn},
		Services:            []*Service{svc},
		ResourceDefinitions: []*Resource{r1, r2},
		QuickstartService:   svc,
		serviceByID:         map[string]*Service{".test.Svc": svc},
		methodByID:          map[string]*Method{},
		messageByID:         map[string]*Message{".test.Msg1": msg1},
		enumByID:            map[string]*Enum{".test.En": en},
		resourceByType:      map[string]*Resource{"test.googleapis.com/R1": r1},
		mapByID:             map[string]*Map{},
		Codec:               "api-codec",
	}
	if diff := cmp.Diff(want, got, allowUnexported()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	a2 := NewAPIBuilderFrom(got).WithName("test2").BuildSyntaxTree()
	if a2.Name != "test2" || len(a2.Messages) != 2 || a2.Message(".test.Msg1") != msg1 {
		t.Errorf("NewAPIBuilderFrom should copy maps and slices: %+v", a2)
	}

	ann := NewAnnotatedAPI("only-codec")
	if ann.Codec != "only-codec" {
		t.Errorf("NewAnnotatedAPI().Codec = %v, want %q", ann.Codec, "only-codec")
	}
}

func TestAPIBuilder_UpdatePackageNameAndBuild(t *testing.T) {
	msg := NewTestMessage("Book").WithPackage("test.v1")
	built, err := NewAPIBuilder().
		WithMessages(msg).
		AddMessage(msg).
		UpdatePackageName().
		WithOverride(ModelOverride{
			Name:        "books",
			Title:       "Books API",
			Description: "Books API docs.",
		}).
		Build()
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}
	if built.PackageName != "test.v1" || built.Name != "books" || built.Title != "Books API" || built.Description != "Books API docs." {
		t.Errorf("unexpected built API metadata: %+v", built)
	}
}
