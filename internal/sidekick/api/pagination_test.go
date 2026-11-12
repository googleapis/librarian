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
)

func TestPageSimple(t *testing.T) {
	resource := NewTestMessage("Resource").WithPackage("package")
	request := NewTestMessage("Request").
		WithPackage("package").
		WithFields(
			NewTestField("parent").WithJSONName("parent").WithType(TypezString),
			NewTestField("page_token").WithJSONName("pageToken").WithType(TypezString),
			NewTestField("page_size").WithJSONName("pageSize").WithType(TypezInt32),
		)
	response := NewTestMessage("Response").
		WithPackage("package").
		WithFields(
			NewTestField("next_page_token").WithJSONName("nextPageToken").WithType(TypezString),
			NewTestField("items").
				WithJSONName("items").
				WithType(TypezMessage).
				WithTypezID(".package.Resource").
				WithRepeated(),
		)
	method := NewTestMethod("List").
		WithID(".package.Service.List").
		WithInputTypeID(".package.Request").
		WithOutputTypeID(".package.Response")
	service := NewTestService("Service").
		WithPackage("package").
		WithMethods(method)
	model := NewTestAPI([]*Message{request, response, resource}, []*Enum{}, []*Service{service})
	UpdateMethodPagination(nil, model)
	if method.Pagination() != request.Fields()[1] {
		t.Errorf("mismatch, want=%v, got=%v", request.Fields()[1], method.Pagination())
	}
	want := NewTestPaginationInfo(response.Fields()[0], response.Fields()[1])
	ignoreParent := cmp.FilterPath(func(p cmp.Path) bool { return p.Last().String() == ".parent" }, cmp.Ignore())
	if diff := cmp.Diff(want, response.Pagination(), allowUnexported(), ignoreParent); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestPageWithOverride(t *testing.T) {
	resource := NewTestMessage("Resource").WithPackage("package")
	request := NewTestMessage("Request").
		WithPackage("package").
		WithFields(
			NewTestField("parent").WithJSONName("parent").WithType(TypezString),
			NewTestField("page_token").WithJSONName("pageToken").WithType(TypezString),
			NewTestField("page_size").WithJSONName("pageSize").WithType(TypezInt32),
		)
	response := NewTestMessage("Response").
		WithPackage("package").
		WithFields(
			NewTestField("next_page_token").WithJSONName("nextPageToken").WithType(TypezString),
			NewTestField("warnings").
				WithJSONName("warnings").
				WithType(TypezMessage).
				WithTypezID(".package.Warning").
				WithRepeated(),
			NewTestField("items").
				WithJSONName("items").
				WithType(TypezMessage).
				WithTypezID(".package.Resource").
				WithRepeated(),
		)
	method := NewTestMethod("List").
		WithID(".package.Service.List").
		WithInputTypeID(".package.Request").
		WithOutputTypeID(".package.Response")
	service := NewTestService("Service").
		WithPackage("package").
		WithMethods(method)
	model := NewTestAPI([]*Message{request, response, resource}, []*Enum{}, []*Service{service})
	overrides := []PaginationOverride{
		{ID: ".package.Service.List", ItemField: "items"},
	}
	UpdateMethodPagination(overrides, model)
	if method.Pagination() != request.Fields()[1] {
		t.Errorf("mismatch, want=%v, got=%v", request.Fields()[1], method.Pagination())
	}
	want := NewTestPaginationInfo(response.Fields()[0], response.Fields()[2])
	ignoreParent := cmp.FilterPath(func(p cmp.Path) bool { return p.Last().String() == ".parent" }, cmp.Ignore())
	if diff := cmp.Diff(want, response.Pagination(), allowUnexported(), ignoreParent); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestPageMissingInputType(t *testing.T) {
	resource := NewTestMessage("Resource").WithPackage("package")
	response := NewTestMessage("Response").
		WithPackage("package").
		WithFields(
			NewTestField("next_page_token").WithJSONName("nextPageToken").WithType(TypezString),
			NewTestField("items").
				WithJSONName("items").
				WithType(TypezMessage).
				WithTypezID(".package.Resource").
				WithRepeated(),
		)
	method := NewTestMethod("List").
		WithID(".package.Service.List").
		WithInputTypeID(".package.Request").
		WithOutputTypeID(".package.Response")
	service := NewTestService("Service").
		WithPackage("package").
		WithMethods(method)
	model := NewTestAPI([]*Message{response, resource}, []*Enum{}, []*Service{service})
	UpdateMethodPagination(nil, model)
	if method.Pagination() != nil {
		t.Errorf("mismatch, want=nil, got=%v", method.Pagination())
	}
}

func TestPageMissingOutputType(t *testing.T) {
	resource := NewTestMessage("Resource").WithPackage("package")
	request := NewTestMessage("Request").
		WithPackage("package").
		WithFields(
			NewTestField("parent").WithJSONName("parent").WithType(TypezString),
			NewTestField("page_token").WithJSONName("pageToken").WithType(TypezString),
			NewTestField("page_size").WithJSONName("pageSize").WithType(TypezInt32),
		)
	method := NewTestMethod("List").
		WithID(".package.Service.List").
		WithInputTypeID(".package.Request").
		WithOutputTypeID(".package.Response")
	service := NewTestService("Service").
		WithPackage("package").
		WithMethods(method)
	model := NewTestAPI([]*Message{request, resource}, []*Enum{}, []*Service{service})
	UpdateMethodPagination(nil, model)
	if method.Pagination() != nil {
		t.Errorf("mismatch, want=nil, got=%v", method.Pagination())
	}
}

func TestPageBadRequest(t *testing.T) {
	resource := NewTestMessage("Resource").WithPackage("package")
	request := NewTestMessage("Request").WithPackage("package")
	response := NewTestMessage("Response").
		WithPackage("package").
		WithFields(
			NewTestField("next_page_token").WithJSONName("nextPageToken").WithType(TypezString),
			NewTestField("items").
				WithJSONName("items").
				WithType(TypezMessage).
				WithTypezID(".package.Resource").
				WithRepeated(),
		)
	method := NewTestMethod("List").
		WithID(".package.Service.List").
		WithInputTypeID(".package.Request").
		WithOutputTypeID(".package.Response")
	service := NewTestService("Service").
		WithPackage("package").
		WithMethods(method)
	model := NewTestAPI([]*Message{request, response, resource}, []*Enum{}, []*Service{service})
	UpdateMethodPagination(nil, model)
	if method.Pagination() != nil {
		t.Errorf("mismatch, want=nil, got=%v", method.Pagination())
	}
}

func TestPageBadResponse(t *testing.T) {
	resource := NewTestMessage("Resource").WithPackage("package")
	request := NewTestMessage("Request").
		WithPackage("package").
		WithFields(
			NewTestField("parent").WithJSONName("parent").WithType(TypezString),
			NewTestField("page_token").WithJSONName("pageToken").WithType(TypezString),
			NewTestField("page_size").WithJSONName("pageSize").WithType(TypezInt32),
		)
	response := NewTestMessage("Response").
		WithPackage("package").
		WithFields(
			NewTestField("parent").WithJSONName("parent").WithType(TypezString),
			NewTestField("page_token").WithJSONName("pageToken").WithType(TypezString),
			NewTestField("page_size").WithJSONName("pageSize").WithType(TypezInt32),
		)
	method := NewTestMethod("List").
		WithID(".package.Service.List").
		WithInputTypeID(".package.Request").
		WithOutputTypeID(".package.Response")
	service := NewTestService("Service").
		WithPackage("package").
		WithMethods(method)
	model := NewTestAPI([]*Message{request, response, resource}, []*Enum{}, []*Service{service})
	UpdateMethodPagination(nil, model)
	if method.Pagination() != nil {
		t.Errorf("mismatch, want=nil, got=%v", method.Pagination())
	}
}

func TestPaginationRequestInfoErrors(t *testing.T) {
	badSize := NewTestMessage("Request").
		WithPackage("package").
		WithFields(
			NewTestField("page_token").WithJSONName("pageToken").WithType(TypezString),
		)
	badToken := NewTestMessage("Request").
		WithPackage("package").
		WithFields(
			NewTestField("page_size").WithJSONName("pageSize").WithType(TypezInt32),
		)

	for _, input := range []*Message{nil, badSize, badToken} {
		if got := paginationRequestInfo(input); got != nil {
			t.Errorf("expected paginationRequestInfo(...) == nil, got=%v, input=%v", got, input)
		}
	}
}

func TestPaginationRequestPageSizeSuccess(t *testing.T) {
	for _, test := range []struct {
		Name    string
		Typez   Typez
		TypezID string
	}{
		{"pageSize", TypezInt32, ""},
		{"pageSize", TypezUint32, ""},
		{"maxResults", TypezInt32, ""},
		{"maxResults", TypezUint32, ""},
		{"maxResults", TypezMessage, ".google.protobuf.Int32Value"},
		{"maxResults", TypezMessage, ".google.protobuf.UInt32Value"},
	} {
		response := NewTestMessage("Response").
			WithPackage("package").
			WithFields(
				NewTestField(test.Name).
					WithJSONName(test.Name).
					WithType(test.Typez).
					WithTypezID(test.TypezID),
			)
		got := paginationRequestPageSize(response)
		ignoreParent := cmp.FilterPath(func(p cmp.Path) bool { return p.Last().String() == ".parent" }, cmp.Ignore())
		if diff := cmp.Diff(response.Fields()[0], got, allowUnexported(), ignoreParent); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestPaginationRequestPageSizeNotMatching(t *testing.T) {
	for _, test := range []struct {
		Name    string
		Typez   Typez
		TypezID string
	}{
		{"badName", TypezInt32, ""},
		{"badName", TypezUint32, ""},
		{"badName", TypezInt32, ""},
		{"badName", TypezUint32, ""},
		{"badName", TypezMessage, ".google.protobuf.Int32Value"},
		{"badName", TypezMessage, ".google.protobuf.UInt32Value"},

		{"pageSize", TypezInt64, ""},
		{"pageSize", TypezUint64, ""},
		{"maxResults", TypezInt64, ""},
		{"maxResults", TypezUint64, ""},
		{"maxResults", TypezMessage, ".google.protobuf.Int64Value"},
		{"maxResults", TypezMessage, ".google.protobuf.UInt64Value"},
	} {
		response := NewTestMessage("Response").
			WithPackage("package").
			WithFields(
				NewTestField(test.Name).
					WithJSONName(test.Name).
					WithType(test.Typez).
					WithTypezID(test.TypezID),
			)
		got := paginationRequestPageSize(response)
		if got != nil {
			t.Errorf("the field should not be a page size, got=%v", got)
		}
	}
}

func TestPaginationRequestToken(t *testing.T) {
	for _, test := range []struct {
		Name  string
		Typez Typez
	}{
		{"badName", TypezString},
		{"nextPageToken", TypezInt32},
	} {
		response := NewTestMessage("Response").
			WithPackage("package").
			WithFields(
				NewTestField(test.Name).
					WithJSONName(test.Name).
					WithType(test.Typez),
			)
		got := paginationRequestToken(response)
		if got != nil {
			t.Errorf("the field should not be a  page token, got=%v", got)
		}
	}
}

func TestPaginationResponseErrors(t *testing.T) {
	badToken := NewTestMessage("Response").
		WithPackage("package").
		WithFields(
			NewTestField("items").
				WithJSONName("items").
				WithType(TypezMessage).
				WithTypezID(".package.Resource").
				WithRepeated(),
		)
	badItems := NewTestMessage("Response").
		WithPackage("package").
		WithFields(
			NewTestField("next_page_token").
				WithJSONName("nextPageToken").
				WithType(TypezString),
		)

	for _, input := range []*Message{badToken, badItems, nil} {
		if got := paginationResponseInfo(nil, ".package.Service.List", input); got != nil {
			t.Errorf("expected paginationResponseInfo(...) == nil, got=%v, input=%v", got, input)
		}
	}
}

func TestPaginationResponseItemMatching(t *testing.T) {
	for _, test := range []struct {
		Repeated bool
		Map      bool
		Typez    Typez
		Name     string
	}{
		{false, true, TypezMessage, "items"},
		{true, false, TypezMessage, "items"},
	} {
		f := NewTestField(test.Name).
			WithJSONName(test.Name).
			WithType(test.Typez)
		if test.Repeated {
			f.WithRepeated()
		}
		if test.Map {
			f.WithMap()
		}
		response := NewTestMessage("Response").
			WithPackage("package").
			WithFields(f)
		got := paginationResponseItem(nil, "package.Service.List", response)
		ignoreParent := cmp.FilterPath(func(p cmp.Path) bool { return p.Last().String() == ".parent" }, cmp.Ignore())
		if diff := cmp.Diff(response.Fields()[0], got, allowUnexported(), ignoreParent); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestPaginationResponseItemMatchingMany(t *testing.T) {
	for _, test := range []struct {
		Repeated bool
		Map      bool
	}{
		{true, false},
		{false, true},
	} {
		f1 := NewTestField("first").WithJSONName("first").WithType(TypezMessage)
		f2 := NewTestField("second").WithJSONName("second").WithType(TypezMessage)
		if test.Repeated {
			f1.WithRepeated()
			f2.WithRepeated()
		}
		if test.Map {
			f1.WithMap()
			f2.WithMap()
		}
		response := NewTestMessage("Response").
			WithPackage("package").
			WithFields(f1, f2)
		got := paginationResponseItem(nil, "package.Service.List", response)
		ignoreParent := cmp.FilterPath(func(p cmp.Path) bool { return p.Last().String() == ".parent" }, cmp.Ignore())
		if diff := cmp.Diff(response.Fields()[0], got, allowUnexported(), ignoreParent); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestPaginationResponseItemMatchingPreferRepeatedOverMap(t *testing.T) {
	response := NewTestMessage("Response").
		WithPackage("package").
		WithFields(
			NewTestField("map").WithJSONName("map").WithType(TypezMessage).WithMap(),
			NewTestField("repeated").WithJSONName("repeated").WithType(TypezMessage).WithRepeated(),
		)
	got := paginationResponseItem(nil, "package.Service.List", response)
	ignoreParent := cmp.FilterPath(func(p cmp.Path) bool { return p.Last().String() == ".parent" }, cmp.Ignore())
	if diff := cmp.Diff(response.Fields()[1], got, allowUnexported(), ignoreParent); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestPaginationResponseItemNotMatching(t *testing.T) {
	overrides := []PaginationOverride{
		{ID: ".package.Service.List", ItemField: "--invalid--"},
	}
	for _, test := range []struct {
		Name      string
		Repeated  bool
		Typez     Typez
		Overrides []PaginationOverride
	}{
		{"badRepeated", false, TypezMessage, nil},
		{"badType", true, TypezString, nil},
		{"bothBad", false, TypezEnum, nil},
		{"badOverride", true, TypezMessage, overrides},
	} {
		f := NewTestField(test.Name).
			WithJSONName(test.Name).
			WithType(test.Typez)
		if test.Repeated {
			f.WithRepeated()
		}
		response := NewTestMessage("Response").
			WithPackage("package").
			WithFields(f)
		got := paginationResponseItem(test.Overrides, ".package.Service.List", response)
		if got != nil {
			t.Errorf("the field should not be a pagination item, got=%v", got)
		}
	}
}

func TestPaginationResponseNextPageToken(t *testing.T) {
	for _, test := range []struct {
		Name  string
		Typez Typez
	}{
		{"badName", TypezString},
		{"nextPageToken", TypezInt32},
	} {
		response := NewTestMessage("Response").
			WithPackage("package").
			WithFields(
				NewTestField(test.Name).
					WithJSONName(test.Name).
					WithType(test.Typez),
			)
		got := paginationResponseNextPageToken(response)
		if got != nil {
			t.Errorf("the field should not be a next page token, got=%v", got)
		}
	}
}

func TestPaginationInfo(t *testing.T) {
	token := NewFieldBuilder().WithName("next_page_token").Build()
	items := NewFieldBuilder().WithName("items").Build()
	got := NewPaginationInfo(token, items)
	want := &PaginationInfo{
		nextPageToken: token,
		pageableItem:  items,
	}
	if diff := cmp.Diff(want, got, allowUnexported()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if got.NextPageToken() != token {
		t.Errorf("NextPageToken() = %v, want %v", got.NextPageToken(), token)
	}
	if got.PageableItem() != items {
		t.Errorf("PageableItem() = %v, want %v", got.PageableItem(), items)
	}
}
