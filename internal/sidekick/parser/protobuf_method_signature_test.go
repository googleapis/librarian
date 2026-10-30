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

package parser

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/api/apitest"
)

func TestProtobuf_Signatures(t *testing.T) {
	requireProtoc(t)
	serviceConfig := &serviceconfig.Service{
		Name:  "secretmanager.googleapis.com",
		Title: "Secret Manager API",
	}

	got, err := makeAPIForProtobuf(serviceConfig, newTestCodeGeneratorRequest(t, "method_signatures.proto"))
	if err != nil {
		t.Fatalf("Failed to make API for Protobuf %v", err)
	}
	if err := api.CrossReference(got); err != nil {
		t.Fatal(err)
	}

	id := ".test.CreateFooRequest"
	gotMessage := got.Message(id)
	if gotMessage == nil {
		t.Fatalf("Cannot find message %s in API State", id)
	}
	var names []string
	for _, f := range gotMessage.Fields {
		names = append(names, f.Name())
	}
	if diff := cmp.Diff([]string{"parent", "foo_id", "flags"}, names); diff != "" {
		t.Fatalf("mismatch (-want +got):\n%s", diff)
	}

	id = ".test.Service.CreateFoo"
	gotMethod := got.Method(id)
	if gotMethod == nil {
		t.Fatalf("Cannot find method %s in API State", id)
	}
	wantSignatures := []*api.MethodSignature{
		api.NewTestMethodSignature("parent", "foo_id", "flags").
			WithFields(gotMessage.Fields...),
		api.NewTestMethodSignature("foo_id", "parent").
			WithFields(gotMessage.Fields[1], gotMessage.Fields[0]),
		api.NewTestMethodSignature("parent", "flags").
			WithFields(gotMessage.Fields[0], gotMessage.Fields[2]),
	}

	// Ignore the MethodSignature.method and Field.Parent back-pointers to avoid recursive cycle testing.
	if diff := cmp.Diff(
		wantSignatures,
		gotMethod.Signatures,
		apitest.AllowUnexported(),
		cmp.FilterPath(func(p cmp.Path) bool { return p.Last().String() == ".method" }, cmp.Ignore()),
		apitest.IgnoreFieldNames("parent")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
