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
	resource := &Message{
		Name: "Resource",
		ID:   ".package.Resource",
	}
	request := &Message{
		Name: "Request",
		ID:   ".package.Request",
		Fields: []*Field{
			{
				Name:     "parent",
				JSONName: "parent",
				ID:       ".package.Request.parent",
				Typez:    STRING_TYPE,
			},
			{
				Name:     "page_token",
				JSONName: "pageToken",
				ID:       ".package.Request.pageToken",
				Typez:    STRING_TYPE,
			},
			{
				Name:     "page_size",
				JSONName: "pageSize",
				ID:       ".package.Request.pageSize",
				Typez:    INT32_TYPE,
			},
		},
	}
	response := &Message{
		Name: "Response",
		ID:   ".package.Response",
		Fields: []*Field{
			{
				Name:     "next_page_token",
				JSONName: "nextPageToken",
				ID:       ".package.Request.nextPageToken",
				Typez:    STRING_TYPE,
			},
			{
				Name:     "items",
				JSONName: "items",
				ID:       ".package.Request.items",
				Typez:    MESSAGE_TYPE,
				TypezID:  ".package.Resource",
				Repeated: true,
			},
		},
	}
	method := &Method{
		Name:         "List",
		ID:           ".package.Service.List",
		InputTypeID:  ".package.Request",
		OutputTypeID: ".package.Response",
	}
	service := &Service{
		Name:    "Service",
		ID:      ".package.Service",
		Methods: []*Method{method},
	}
	model := NewTestAPI([]*Message{request, response, resource}, []*Enum{}, []*Service{service})
	UpdateMethodPagination(nil, model)
	if method.Pagination != request.Fields[1] {
		t.Errorf("mismatch, want=%v, got=%v", request.Fields[1], method.Pagination)
	}
	want := &PaginationInfo{
		NextPageToken: response.Fields[0],
		PageableItem:  response.Fields[1],
	}
	if diff := cmp.Diff(want, response.Pagination); diff != "" {
		t.Errorf("mismatch, (-want, +got):\n%s", diff)
	}
}

func TestPageWithOverride(t *testing.T) {
	resource := &Message{
		Name: "Resource",
		ID:   ".package.Resource",
	}
	request := &Message{
		Name: "Request",
		ID:   ".package.Request",
		Fields: []*Field{
			{
				Name:     "parent",
				JSONName: "parent",
				ID:       ".package.Request.parent",
				Typez:    STRING_TYPE,
			},
			{
				Name:     "page_token",
				JSONName: "pageToken",
				ID:       ".package.Request.pageToken",
				Typez:    STRING_TYPE,
			},
			{
				Name:     "page_size",
				JSONName: "pageSize",
				ID:       ".package.Request.pageSize",
				Typez:    INT32_TYPE,
			},
		},
	}
	response := &Message{
		Name: "Response",
		ID:   ".package.Response",
		Fields: []*Field{
			{
				Name:     "next_page_token",
				JSONName: "nextPageToken",
				ID:       ".package.Request.nextPageToken",
				Typez:    STRING_TYPE,
			},
			{
				Name:     "warnings",
				JSONName: "warnings",
				ID:       ".package.Request.warnings",
				Typez:    MESSAGE_TYPE,
				TypezID:  ".package.Warning",
				Repeated: true,
			},
			{
				Name:     "items",
				JSONName: "items",
				ID:       ".package.Request.items",
				Typez:    MESSAGE_TYPE,
				TypezID:  ".package.Resource",
				Repeated: true,
			},
		},
	}
	method := &Method{
		Name:         "List",
		ID:           ".package.Service.List",
		InputTypeID:  ".package.Request",
		OutputTypeID: ".package.Response",
	}
	service := &Service{
		Name:    "Service",
		ID:      ".package.Service",
		Methods: []*Method{method},
	}
	model := NewTestAPI([]*Message{request, response, resource}, []*Enum{}, []*Service{service})
	overrides := []PaginationOverride{
		{ID: ".package.Service.List", ItemField: "items"},
	}
	UpdateMethodPagination(overrides, model)
	if method.Pagination != request.Fields[1] {
		t.Errorf("mismatch, want=%v, got=%v", request.Fields[1], method.Pagination)
	}
	want := &PaginationInfo{
		NextPageToken: response.Fields[0],
		PageableItem:  response.Fields[2],
	}
	if diff := cmp.Diff(want, response.Pagination); diff != "" {
		t.Errorf("mismatch, (-want, +got):\n%s", diff)
	}
}

func TestPageMissingInputType(t *testing.T) {
	resource := &Message{
		Name: "Resource",
		ID:   ".package.Resource",
	}
	response := &Message{
		Name: "Response",
		ID:   ".package.Response",
		Fields: []*Field{
			{
				Name:     "next_page_token",
				JSONName: "nextPageToken",
				ID:       ".package.Request.nextPageToken",
				Typez:    STRING_TYPE,
			},
			{
				Name:     "items",
				JSONName: "items",
				ID:       ".package.Request.items",
				Typez:    MESSAGE_TYPE,
				TypezID:  ".package.Resource",
				Repeated: true,
			},
		},
	}
	method := &Method{
		Name:         "List",
		ID:           ".package.Service.List",
		InputTypeID:  ".package.Request",
		OutputTypeID: ".package.Response",
	}
	service := &Service{
		Name:    "Service",
		ID:      ".package.Service",
		Methods: []*Method{method},
	}
	model := NewTestAPI([]*Message{response, resource}, []*Enum{}, []*Service{service})
	UpdateMethodPagination(nil, model)
	if method.Pagination != nil {
		t.Errorf("mismatch, want=nil, got=%v", method.Pagination)
	}
}

func TestPageMissingOutputType(t *testing.T) {
	resource := &Message{
		Name: "Resource",
		ID:   ".package.Resource",
	}
	request := &Message{
		Name: "Request",
		ID:   ".package.Request",
		Fields: []*Field{
			{
				Name:     "parent",
				JSONName: "parent",
				ID:       ".package.Request.parent",
				Typez:    STRING_TYPE,
			},
			{
				Name:     "page_token",
				JSONName: "pageToken",
				ID:       ".package.Request.pageToken",
				Typez:    STRING_TYPE,
			},
			{
				Name:     "page_size",
				JSONName: "pageSize",
				ID:       ".package.Request.pageSize",
				Typez:    INT32_TYPE,
			},
		},
	}
	method := &Method{
		Name:         "List",
		ID:           ".package.Service.List",
		InputTypeID:  ".package.Request",
		OutputTypeID: ".package.Response",
	}
	service := &Service{
		Name:    "Service",
		ID:      ".package.Service",
		Methods: []*Method{method},
	}
	model := NewTestAPI([]*Message{request, resource}, []*Enum{}, []*Service{service})
	UpdateMethodPagination(nil, model)
	if method.Pagination != nil {
		t.Errorf("mismatch, want=nil, got=%v", method.Pagination)
	}
}

func TestPageBadRequest(t *testing.T) {
	resource := &Message{
		Name: "Resource",
		ID:   ".package.Resource",
	}
	request := &Message{
		Name:   "Request",
		ID:     ".package.Request",
		Fields: []*Field{},
	}
	response := &Message{
		Name: "Response",
		ID:   ".package.Response",
		Fields: []*Field{
			{
				Name:     "next_page_token",
				JSONName: "nextPageToken",
				ID:       ".package.Request.nextPageToken",
				Typez:    STRING_TYPE,
			},
			{
				Name:     "items",
				JSONName: "items",
				ID:       ".package.Request.items",
				Typez:    MESSAGE_TYPE,
				TypezID:  ".package.Resource",
				Repeated: true,
			},
		},
	}
	method := &Method{
		Name:         "List",
		ID:           ".package.Service.List",
		InputTypeID:  ".package.Request",
		OutputTypeID: ".package.Response",
	}
	service := &Service{
		Name:    "Service",
		ID:      ".package.Service",
		Methods: []*Method{method},
	}
	model := NewTestAPI([]*Message{request, response, resource}, []*Enum{}, []*Service{service})
	UpdateMethodPagination(nil, model)
	if method.Pagination != nil {
		t.Errorf("mismatch, want=nil, got=%v", method.Pagination)
	}
}

func TestPageBadResponse(t *testing.T) {
	resource := &Message{
		Name: "Resource",
		ID:   ".package.Resource",
	}
	request := &Message{
		Name:   "Request",
		ID:     ".package.Request",
		Fields: []*Field{},
	}
	response := &Message{
		Name: "Response",
		ID:   ".package.Response",
		Fields: []*Field{
			{
				Name:     "parent",
				JSONName: "parent",
				ID:       ".package.Request.parent",
				Typez:    STRING_TYPE,
			},
			{
				Name:     "page_token",
				JSONName: "pageToken",
				ID:       ".package.Request.pageToken",
				Typez:    STRING_TYPE,
			},
			{
				Name:     "page_size",
				JSONName: "pageSize",
				ID:       ".package.Request.pageSize",
				Typez:    INT32_TYPE,
			},
		},
	}
	method := &Method{
		Name:         "List",
		ID:           ".package.Service.List",
		InputTypeID:  ".package.Request",
		OutputTypeID: ".package.Response",
	}
	service := &Service{
		Name:    "Service",
		ID:      ".package.Service",
		Methods: []*Method{method},
	}
	model := NewTestAPI([]*Message{request, response, resource}, []*Enum{}, []*Service{service})
	UpdateMethodPagination(nil, model)
	if method.Pagination != nil {
		t.Errorf("mismatch, want=nil, got=%v", method.Pagination)
	}
}

func TestPaginationRequestInfoErrors(t *testing.T) {
	badSize := &Message{
		Name: "Request",
		ID:   ".package.Request",
		Fields: []*Field{
			{
				Name:     "page_token",
				JSONName: "pageToken",
				ID:       ".package.Request.pageToken",
				Typez:    STRING_TYPE,
			},
		},
	}
	badToken := &Message{
		Name: "Request",
		ID:   ".package.Request",
		Fields: []*Field{
			{
				Name:     "page_size",
				JSONName: "pageSize",
				ID:       ".package.Request.pageSize",
				Typez:    INT32_TYPE,
			},
		},
	}

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
		{"pageSize", INT32_TYPE, ""},
		{"pageSize", UINT32_TYPE, ""},
		{"maxResults", INT32_TYPE, ""},
		{"maxResults", UINT32_TYPE, ""},
		{"maxResults", MESSAGE_TYPE, ".google.protobuf.Int32Value"},
		{"maxResults", MESSAGE_TYPE, ".google.protobuf.UInt32Value"},
	} {
		response := &Message{
			Name: "Response",
			ID:   ".package.Response",
			Fields: []*Field{
				{
					Name:     test.Name,
					JSONName: test.Name,
					Typez:    test.Typez,
					TypezID:  test.TypezID,
				},
			},
		}
		got := paginationRequestPageSize(response)
		if diff := cmp.Diff(response.Fields[0], got); diff != "" {
			t.Errorf("mismatch (-want, +got):\n%s", diff)
		}
	}
}

func TestPaginationRequestPageSizeNotMatching(t *testing.T) {
	for _, test := range []struct {
		Name    string
		Typez   Typez
		TypezID string
	}{
		{"badName", INT32_TYPE, ""},
		{"badName", UINT32_TYPE, ""},
		{"badName", INT32_TYPE, ""},
		{"badName", UINT32_TYPE, ""},
		{"badName", MESSAGE_TYPE, ".google.protobuf.Int32Value"},
		{"badName", MESSAGE_TYPE, ".google.protobuf.UInt32Value"},

		{"pageSize", INT64_TYPE, ""},
		{"pageSize", UINT64_TYPE, ""},
		{"maxResults", INT64_TYPE, ""},
		{"maxResults", UINT64_TYPE, ""},
		{"maxResults", MESSAGE_TYPE, ".google.protobuf.Int64Value"},
		{"maxResults", MESSAGE_TYPE, ".google.protobuf.UInt64Value"},
	} {
		response := &Message{
			Name: "Response",
			ID:   ".package.Response",
			Fields: []*Field{
				{
					Name:     test.Name,
					JSONName: test.Name,
					Typez:    test.Typez,
					TypezID:  test.TypezID,
				},
			},
		}
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
		{"badName", STRING_TYPE},
		{"nextPageToken", INT32_TYPE},
	} {
		response := &Message{
			Name: "Response",
			ID:   ".package.Response",
			Fields: []*Field{
				{
					Name:     test.Name,
					JSONName: test.Name,
					Typez:    test.Typez,
				},
			},
		}
		got := paginationRequestToken(response)
		if got != nil {
			t.Errorf("the field should not be a  page token, got=%v", got)
		}
	}
}

func TestPaginationResponseErrors(t *testing.T) {
	badToken := &Message{
		Name: "Response",
		ID:   ".package.Response",
		Fields: []*Field{
			{
				Name:     "items",
				JSONName: "items",
				ID:       ".package.Request.items",
				Typez:    MESSAGE_TYPE,
				TypezID:  ".package.Resource",
				Repeated: true,
			},
		},
	}
	badItems := &Message{
		Name: "Response",
		ID:   ".package.Response",
		Fields: []*Field{
			{
				Name:     "next_page_token",
				JSONName: "nextPageToken",
				ID:       ".package.Request.nextPageToken",
				Typez:    STRING_TYPE,
			},
		},
	}

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
		{false, true, MESSAGE_TYPE, "items"},
		{true, false, MESSAGE_TYPE, "items"},
	} {
		response := &Message{
			Name: "Response",
			ID:   ".package.Response",
			Fields: []*Field{
				{
					Name:     test.Name,
					JSONName: test.Name,
					Typez:    test.Typez,
					Repeated: test.Repeated,
					Map:      test.Map,
				},
			},
		}
		got := paginationResponseItem(nil, "package.Service.List", response)
		if diff := cmp.Diff(response.Fields[0], got); diff != "" {
			t.Errorf("mismatch (-want, +got):\n%s", diff)
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
		response := &Message{
			Name: "Response",
			ID:   ".package.Response",
			Fields: []*Field{
				{
					Name:     "first",
					JSONName: "first",
					Typez:    MESSAGE_TYPE,
					Repeated: test.Repeated,
					Map:      test.Map,
				},
				{
					Name:     "second",
					JSONName: "second",
					Typez:    MESSAGE_TYPE,
					Repeated: test.Repeated,
					Map:      test.Map,
				},
			},
		}
		got := paginationResponseItem(nil, "package.Service.List", response)
		if diff := cmp.Diff(response.Fields[0], got); diff != "" {
			t.Errorf("mismatch (-want, +got):\n%s", diff)
		}
	}
}

func TestPaginationResponseItemMatchingPreferRepeatedOverMap(t *testing.T) {
	response := &Message{
		Name: "Response",
		ID:   ".package.Response",
		Fields: []*Field{
			{
				Name:     "map",
				JSONName: "map",
				Typez:    MESSAGE_TYPE,
				Map:      true,
			},
			{
				Name:     "repeated",
				JSONName: "repeated",
				Typez:    MESSAGE_TYPE,
				Repeated: true,
			},
		},
	}
	got := paginationResponseItem(nil, "package.Service.List", response)
	if diff := cmp.Diff(response.Fields[1], got); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
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
		{"badRepeated", false, MESSAGE_TYPE, nil},
		{"badType", true, STRING_TYPE, nil},
		{"bothBad", false, ENUM_TYPE, nil},
		{"badOverride", true, MESSAGE_TYPE, overrides},
	} {
		response := &Message{
			Name: "Response",
			ID:   ".package.Response",
			Fields: []*Field{
				{
					Name:     test.Name,
					JSONName: test.Name,
					Typez:    test.Typez,
					Repeated: test.Repeated,
				},
			},
		}
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
		{"badName", STRING_TYPE},
		{"nextPageToken", INT32_TYPE},
	} {
		response := &Message{
			Name: "Response",
			ID:   ".package.Response",
			Fields: []*Field{
				{
					Name:     test.Name,
					JSONName: test.Name,
					Typez:    test.Typez,
				},
			},
		}
		got := paginationResponseNextPageToken(response)
		if got != nil {
			t.Errorf("the field should not be a next page token, got=%v", got)
		}
	}
}
