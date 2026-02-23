// Copyright 2024 Google LLC
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
)

const (
	pageSize      = "pageSize"
	maxResults    = "maxResults"
	pageToken     = "pageToken"
	nextPageToken = "nextPageToken"
)

// PaginationOverride describes overrides for pagination config of a method.
type PaginationOverride struct {
	// The method ID.
	ID string
	// The name of the field used for `items`.
	ItemField string
}

// PaginationInfo contains information related to pagination aka [AIP-4233](https://google.aip.dev/client-libraries/4233).
type PaginationInfo struct {
	// The field that gives us the next page token.
	NextPageToken *Field
	// PageableItem is the field to be paginated over.
	PageableItem *Field
}

// UpdateMethodPagination marks all methods that conform to
// [AIP-4233](https://google.aip.dev/client-libraries/4233) as pageable.
func UpdateMethodPagination(overrides []PaginationOverride, a *API) {
	for _, m := range a.State.MethodByID {
		reqMsg := a.State.MessageByID[m.InputTypeID]
		pageTokenField := paginationRequestInfo(reqMsg)
		if pageTokenField == nil {
			continue
		}

		respMsg := a.State.MessageByID[m.OutputTypeID]
		paginationInfo := paginationResponseInfo(overrides, m.ID, respMsg)
		if paginationInfo == nil {
			continue
		}
		m.Pagination = pageTokenField
		respMsg.Pagination = paginationInfo
	}
}

func paginationRequestInfo(request *Message) *Field {
	if request == nil {
		return nil
	}
	if pageSizeField := paginationRequestPageSize(request); pageSizeField == nil {
		return nil
	}
	return paginationRequestToken(request)
}

func paginationRequestPageSize(request *Message) *Field {
	for _, field := range request.Fields {
		// Some legacy services (e.g. sqladmin.googleapis.com)
		// predate AIP-4233 and use `maxResults` instead of
		// `pageSize` for the field name.
		if field.JSONName == pageSize && isPaginationPageSizeType(field) {
			return field
		}
		if field.JSONName == maxResults && isPaginationMaxResultsType(field) {
			return field
		}
	}
	return nil
}

func isPaginationPageSizeType(field *Field) bool {
	return field.Typez == INT32_TYPE || field.Typez == UINT32_TYPE
}

func isPaginationMaxResultsType(field *Field) bool {
	// Legacy maxResults types can be int32/uint32, and protobuf wrappers Int32Value/UInt32Value.
	if isPaginationPageSizeType(field) {
		return true
	}
	return field.Typez == MESSAGE_TYPE &&
		(field.TypezID == ".google.protobuf.Int32Value" ||
			field.TypezID == ".google.protobuf.UInt32Value")
}

func paginationRequestToken(request *Message) *Field {
	for _, field := range request.Fields {
		if field.JSONName == pageToken && field.Typez == STRING_TYPE {
			return field
		}
	}
	return nil
}

func paginationResponseInfo(overrides []PaginationOverride, methodID string, response *Message) *PaginationInfo {
	if response == nil {
		return nil
	}
	pageableItem := paginationResponseItem(overrides, methodID, response)
	nextPageToken := paginationResponseNextPageToken(response)
	if pageableItem == nil || nextPageToken == nil {
		return nil
	}
	return &PaginationInfo{
		PageableItem:  pageableItem,
		NextPageToken: nextPageToken,
	}
}

func paginationResponseItem(overrides []PaginationOverride, methodID string, response *Message) *Field {
	idx := slices.IndexFunc(overrides, func(o PaginationOverride) bool { return o.ID == methodID })
	if idx != -1 {
		overrideName := overrides[idx].ItemField
		fieldIdx := slices.IndexFunc(response.Fields, func(f *Field) bool { return f.Name == overrideName })
		if fieldIdx == -1 {
			return nil
		}
		return response.Fields[fieldIdx]
	}

	var mapItems *Field
	for _, field := range response.Fields {
		if field.Map && mapItems == nil {
			mapItems = field
		}
		if field.Repeated && field.Typez == MESSAGE_TYPE {
			return field
		}
	}
	return mapItems
}

func paginationResponseNextPageToken(response *Message) *Field {
	for _, field := range response.Fields {
		if field.JSONName == nextPageToken && field.Typez == STRING_TYPE {
			return field
		}
	}
	return nil
}
