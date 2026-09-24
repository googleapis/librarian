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
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestFindServiceDependencies(t *testing.T) {
	someEnum := NewTestEnum("SomeEnum")
	msg := NewTestMessage("Message")
	msg.WithFields(
		NewTestField("a").WithType(TypezString),
		NewTestField("b").WithEnumType(someEnum),
		NewTestField("c").WithMessageType(msg).WithOptional(),
	)
	unused := NewTestMessage("Unused")
	request := NewTestMessage("Request").
		WithFields(NewTestField("body").WithMessageType(msg))
	response := NewTestMessage("Response")
	empty := NewTestMessage("Empty")
	opMetadata := NewTestMessage("OpMetadata")
	opResponse := NewTestMessage("OpResponse")

	enums := []*Enum{someEnum}
	messages := []*Message{msg, unused, request, response, empty, opMetadata, opResponse}
	services := []*Service{
		NewTestService("Service1").WithMethods(
			NewTestMethod("Method0").
				WithInput(request).
				WithOutput(response),
		),
		NewTestService("Service2").WithMethods(
			NewTestMethod("Method0").
				WithInput(empty).
				WithOutput(empty).
				WithOperationInfo(&OperationInfo{
					MetadataTypeID: opMetadata.ID,
					ResponseTypeID: opResponse.ID,
				}),
		),
	}
	less := func(a, b string) bool { return a < b }
	model := NewTestAPI(messages, enums, services)
	CrossReference(model)
	got := FindServiceDependencies(model, ".test.NotFound")
	want := &ServiceDependencies{}
	if diff := cmp.Diff(want, got, cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	got = FindServiceDependencies(model, ".test.Service1")
	want = &ServiceDependencies{
		Messages: []string{".test.Request", ".test.Response", ".test.Message"},
		Enums:    []string{".test.SomeEnum"},
	}
	if diff := cmp.Diff(want, got, cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	got = FindServiceDependencies(model, ".test.Service2")
	want = &ServiceDependencies{
		Messages: []string{".test.Empty", ".test.OpMetadata", ".test.OpResponse"},
	}
	if diff := cmp.Diff(want, got, cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
