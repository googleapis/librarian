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

package parser

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/internal/api"
	"github.com/googleapis/librarian/internal/sidekick/internal/config"
)

func TestPageSimple(t *testing.T) {
	resource := &api.Message{
		Name: "Resource",
		ID:   ".package.Resource",
	}
	request := &api.Message{
		Name: "Request",
		ID:   ".package.Request",
		Fields: []*api.Field{
			{
				Name:     "parent",
				JSONName: "parent",
				ID:       ".package.Request.parent",
				Typez:    api.STRING_TYPE,
			},
			{
				Name:     "page_token",
				JSONName: "pageToken",
				ID:       ".package.Request.pageToken",
				Typez:    api.STRING_TYPE,
			},
			{
				Name:     "page_size",
				JSONName: "pageSize",
				ID:       ".package.Request.pageSize",
				Typez:    api.INT32_TYPE,
			},
		},
	}
	response := &api.Message{
		Name: "Response",
		ID:   ".package.Response",
		Fields: []*api.Field{
			{
				Name:     "next_page_token",
				JSONName: "nextPageToken",
				ID:       ".package.Request.nextPageToken",
				Typez:    api.STRING_TYPE,
			},
			{
				Name:     "items",
				JSONName: "items",
				ID:       ".package.Request.items",
				Typez:    api.MESSAGE_TYPE,
				TypezID:  ".package.Resource",
				Repeated: true,
			},
		},
	}
	method := &api.Method{
		Name:         "List",
		ID:           ".package.Service.List",
		InputTypeID:  ".package.Request",
		OutputTypeID: ".package.Response",
	}
	service := &api.Service{
		Name:    "Service",
		ID:      ".package.Service",
		Methods: []*api.Method{method},
	}
	model := api.NewTestAPI([]*api.Message{request, response, resource}, []*api.Enum{}, []*api.Service{service})
	updateMethodPagination(nil, model)
	if method.Pagination != request.Fields[1] {
		t.Errorf("mismatch, want=%v, got=%v", request.Fields[1], method.Pagination)
	}
	want := &api.PaginationInfo{
		NextPageToken: response.Fields[0],
		PageableItem:  response.Fields[1],
	}
	if diff := cmp.Diff(want, response.Pagination); diff != "" {
		t.Errorf("mismatch, (-want, +got):\n%s", diff)
	}
}

func TestPageWithOverride(t *testing.T) {
	resource := &api.Message{
		Name: "Resource",
		ID:   ".package.Resource",
	}
	request := &api.Message{
		Name: "Request",
		ID:   ".package.Request",
		Fields: []*api.Field{
			{
				Name:     "parent",
				JSONName: "parent",
				ID:       ".package.Request.parent",
				Typez:    api.STRING_TYPE,
			},
			{
				Name:     "page_token",
				JSONName: "pageToken",
				ID:       ".package.Request.pageToken",
				Typez:    api.STRING_TYPE,
			},
			{
				Name:     "page_size",
				JSONName: "pageSize",
				ID:       ".package.Request.pageSize",
				Typez:    api.INT32_TYPE,
			},
		},
	}
	response := &api.Message{
		Name: "Response",
		ID:   ".package.Response",
		Fields: []*api.Field{
			{
				Name:     "next_page_token",
				JSONName: "nextPageToken",
				ID:       ".package.Request.nextPageToken",
				Typez:    api.STRING_TYPE,
			},
			{
				Name:     "warnings",
				JSONName: "warnings",
				ID:       ".package.Request.warnings",
				Typez:    api.MESSAGE_TYPE,
				TypezID:  ".package.Warning",
				Repeated: true,
			},
			{
				Name:     "items",
				JSONName: "items",
				ID:       ".package.Request.items",
				Typez:    api.MESSAGE_TYPE,
				TypezID:  ".package.Resource",
				Repeated: true,
			},
		},
	}
	method := &api.Method{
		Name:         "List",
		ID:           ".package.Service.List",
		InputTypeID:  ".package.Request",
		OutputTypeID: ".package.Response",
	}
	service := &api.Service{
		Name:    "Service",
		ID:      ".package.Service",
		Methods: []*api.Method{method},
	}
	model := api.NewTestAPI([]*api.Message{request, response, resource}, []*api.Enum{}, []*api.Service{service})
	overrides := []config.PaginationOverride{
		{ID: ".package.Service.List", ItemField: "items"},
	}
	updateMethodPagination(overrides, model)
	if method.Pagination != request.Fields[1] {
		t.Errorf("mismatch, want=%v, got=%v", request.Fields[1], method.Pagination)
	}
	want := &api.PaginationInfo{
		NextPageToken: response.Fields[0],
		PageableItem:  response.Fields[2],
	}
	if diff := cmp.Diff(want, response.Pagination); diff != "" {
		t.Errorf("mismatch, (-want, +got):\n%s", diff)
	}
}

func TestPageMissingInputType(t *testing.T) {
	resource := &api.Message{
		Name: "Resource",
		ID:   ".package.Resource",
	}
	response := &api.Message{
		Name: "Response",
		ID:   ".package.Response",
		Fields: []*api.Field{
			{
				Name:     "next_page_token",
				JSONName: "nextPageToken",
				ID:       ".package.Request.nextPageToken",
				Typez:    api.STRING_TYPE,
			},
			{
				Name:     "items",
				JSONName: "items",
				ID:       ".package.Request.items",
				Typez:    api.MESSAGE_TYPE,
				TypezID:  ".package.Resource",
				Repeated: true,
			},
		},
	}
	method := &api.Method{
		Name:         "List",
		ID:           ".package.Service.List",
		InputTypeID:  ".package.Request",
		OutputTypeID: ".package.Response",
	}
	service := &api.Service{
		Name:    "Service",
		ID:      ".package.Service",
		Methods: []*api.Method{method},
	}
	model := api.NewTestAPI([]*api.Message{response, resource}, []*api.Enum{}, []*api.Service{service})
	updateMethodPagination(nil, model)
	if method.Pagination != nil {
		t.Errorf("mismatch, want=nil, got=%v", method.Pagination)
	}
}

func TestPageMissingOutputType(t *testing.T) {
	resource := &api.Message{
		Name: "Resource",
		ID:   ".package.Resource",
	}
	request := &api.Message{
		Name: "Request",
		ID:   ".package.Request",
		Fields: []*api.Field{
			{
				Name:     "parent",
				JSONName: "parent",
				ID:       ".package.Request.parent",
				Typez:    api.STRING_TYPE,
			},
			{
				Name:     "page_token",
				JSONName: "pageToken",
				ID:       ".package.Request.pageToken",
				Typez:    api.STRING_TYPE,
			},
			{
				Name:     "page_size",
				JSONName: "pageSize",
				ID:       ".package.Request.pageSize",
				Typez:    api.INT32_TYPE,
			},
		},
	}
	method := &api.Method{
		Name:         "List",
		ID:           ".package.Service.List",
		InputTypeID:  ".package.Request",
		OutputTypeID: ".package.Response",
	}
	service := &api.Service{
		Name:    "Service",
		ID:      ".package.Service",
		Methods: []*api.Method{method},
	}
	model := api.NewTestAPI([]*api.Message{request, resource}, []*api.Enum{}, []*api.Service{service})
	updateMethodPagination(nil, model)
	if method.Pagination != nil {
		t.Errorf("mismatch, want=nil, got=%v", method.Pagination)
	}
}

func TestPageBadRequest(t *testing.T) {
	resource := &api.Message{
		Name: "Resource",
		ID:   ".package.Resource",
	}
	request := &api.Message{
		Name:   "Request",
		ID:     ".package.Request",
		Fields: []*api.Field{},
	}
	response := &api.Message{
		Name: "Response",
		ID:   ".package.Response",
		Fields: []*api.Field{
			{
				Name:     "next_page_token",
				JSONName: "nextPageToken",
				ID:       ".package.Request.nextPageToken",
				Typez:    api.STRING_TYPE,
			},
			{
				Name:     "items",
				JSONName: "items",
				ID:       ".package.Request.items",
				Typez:    api.MESSAGE_TYPE,
				TypezID:  ".package.Resource",
				Repeated: true,
			},
		},
	}
	method := &api.Method{
		Name:         "List",
		ID:           ".package.Service.List",
		InputTypeID:  ".package.Request",
		OutputTypeID: ".package.Response",
	}
	service := &api.Service{
		Name:    "Service",
		ID:      ".package.Service",
		Methods: []*api.Method{method},
	}
	model := api.NewTestAPI([]*api.Message{request, response, resource}, []*api.Enum{}, []*api.Service{service})
	updateMethodPagination(nil, model)
	if method.Pagination != nil {
		t.Errorf("mismatch, want=nil, got=%v", method.Pagination)
	}
}

func TestPageBadResponse(t *testing.T) {
	resource := &api.Message{
		Name: "Resource",
		ID:   ".package.Resource",
	}
	request := &api.Message{
		Name:   "Request",
		ID:     ".package.Request",
		Fields: []*api.Field{},
	}
	response := &api.Message{
		Name: "Response",
		ID:   ".package.Response",
		Fields: []*api.Field{
			{
				Name:     "parent",
				JSONName: "parent",
				ID:       ".package.Request.parent",
				Typez:    api.STRING_TYPE,
			},
			{
				Name:     "page_token",
				JSONName: "pageToken",
				ID:       ".package.Request.pageToken",
				Typez:    api.STRING_TYPE,
			},
			{
				Name:     "page_size",
				JSONName: "pageSize",
				ID:       ".package.Request.pageSize",
				Typez:    api.INT32_TYPE,
			},
		},
	}
	method := &api.Method{
		Name:         "List",
		ID:           ".package.Service.List",
		InputTypeID:  ".package.Request",
		OutputTypeID: ".package.Response",
	}
	service := &api.Service{
		Name:    "Service",
		ID:      ".package.Service",
		Methods: []*api.Method{method},
	}
	model := api.NewTestAPI([]*api.Message{request, response, resource}, []*api.Enum{}, []*api.Service{service})
	updateMethodPagination(nil, model)
	if method.Pagination != nil {
		t.Errorf("mismatch, want=nil, got=%v", method.Pagination)
	}
}

func TestPaginationRequestInfoErrors(t *testing.T) {
	badSize := &api.Message{
		Name: "Request",
		ID:   ".package.Request",
		Fields: []*api.Field{
			{
				Name:     "page_token",
				JSONName: "pageToken",
				ID:       ".package.Request.pageToken",
				Typez:    api.STRING_TYPE,
			},
		},
	}
	badToken := &api.Message{
		Name: "Request",
		ID:   ".package.Request",
		Fields: []*api.Field{
			{
				Name:     "page_size",
				JSONName: "pageSize",
				ID:       ".package.Request.pageSize",
				Typez:    api.INT32_TYPE,
			},
		},
	}

	for _, input := range []*api.Message{nil, badSize, badToken} {
		if got := paginationRequestInfo(input); got != nil {
			t.Errorf("expected paginationRequestInfo(...) == nil, got=%v, input=%v", got, input)
		}
	}
}

func TestPaginationRequestPageSizeSuccess(t *testing.T) {
	for _, test := range []struct {
		Name    string
		Typez   api.Typez
		TypezID string
	}{
		{"pageSize", api.INT32_TYPE, ""},
		{"pageSize", api.UINT32_TYPE, ""},
		{"maxResults", api.INT32_TYPE, ""},
		{"maxResults", api.UINT32_TYPE, ""},
		{"maxResults", api.MESSAGE_TYPE, ".google.protobuf.Int32Value"},
		{"maxResults", api.MESSAGE_TYPE, ".google.protobuf.UInt32Value"},
	} {
		response := &api.Message{
			Name: "Response",
			ID:   ".package.Response",
			Fields: []*api.Field{
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
		Typez   api.Typez
		TypezID string
	}{
		{"badName", api.INT32_TYPE, ""},
		{"badName", api.UINT32_TYPE, ""},
		{"badName", api.INT32_TYPE, ""},
		{"badName", api.UINT32_TYPE, ""},
		{"badName", api.MESSAGE_TYPE, ".google.protobuf.Int32Value"},
		{"badName", api.MESSAGE_TYPE, ".google.protobuf.UInt32Value"},

		{"pageSize", api.INT64_TYPE, ""},
		{"pageSize", api.UINT64_TYPE, ""},
		{"maxResults", api.INT64_TYPE, ""},
		{"maxResults", api.UINT64_TYPE, ""},
		{"maxResults", api.MESSAGE_TYPE, ".google.protobuf.Int64Value"},
		{"maxResults", api.MESSAGE_TYPE, ".google.protobuf.UInt64Value"},
	} {
		response := &api.Message{
			Name: "Response",
			ID:   ".package.Response",
			Fields: []*api.Field{
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
		Typez api.Typez
	}{
		{"badName", api.STRING_TYPE},
		{"nextPageToken", api.INT32_TYPE},
	} {
		response := &api.Message{
			Name: "Response",
			ID:   ".package.Response",
			Fields: []*api.Field{
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
	badToken := &api.Message{
		Name: "Response",
		ID:   ".package.Response",
		Fields: []*api.Field{
			{
				Name:     "items",
				JSONName: "items",
				ID:       ".package.Request.items",
				Typez:    api.MESSAGE_TYPE,
				TypezID:  ".package.Resource",
				Repeated: true,
			},
		},
	}
	badItems := &api.Message{
		Name: "Response",
		ID:   ".package.Response",
		Fields: []*api.Field{
			{
				Name:     "next_page_token",
				JSONName: "nextPageToken",
				ID:       ".package.Request.nextPageToken",
				Typez:    api.STRING_TYPE,
			},
		},
	}

	for _, input := range []*api.Message{badToken, badItems, nil} {
		if got := paginationResponseInfo(nil, ".package.Service.List", input); got != nil {
			t.Errorf("expected paginationResponseInfo(...) == nil, got=%v, input=%v", got, input)
		}
	}
}

func TestPaginationResponseItem(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Repeated bool
		Typez    api.Typez
	}{
		{"badRepeated", false, api.MESSAGE_TYPE},
		{"badType", true, api.STRING_TYPE},
		{"bothBad", false, api.ENUM_TYPE},
	} {
		response := &api.Message{
			Name: "Response",
			ID:   ".package.Response",
			Fields: []*api.Field{
				{
					Name:     test.Name,
					JSONName: test.Name,
					Typez:    test.Typez,
					Repeated: test.Repeated,
				},
			},
		}
		got := paginationResponseItem(nil, ".package.Service.List", response)
		if got != nil {
			t.Errorf("the field should not be a pagination item, got=%v", got)
		}
	}
}

func TestPaginationResponseNextPageToken(t *testing.T) {
	for _, test := range []struct {
		Name  string
		Typez api.Typez
	}{
		{"badName", api.STRING_TYPE},
		{"nextPageToken", api.INT32_TYPE},
	} {
		response := &api.Message{
			Name: "Response",
			ID:   ".package.Response",
			Fields: []*api.Field{
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
