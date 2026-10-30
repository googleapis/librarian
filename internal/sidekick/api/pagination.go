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
	nextPageToken *Field
	pageableItem  *Field
}

// NewPaginationInfo creates a new [PaginationInfo].
func NewPaginationInfo(nextPageToken, pageableItem *Field) *PaginationInfo {
	return &PaginationInfo{
		nextPageToken: nextPageToken,
		pageableItem:  pageableItem,
	}
}

// NextPageToken returns the field that gives us the next page token.
func (p *PaginationInfo) NextPageToken() *Field {
	return p.nextPageToken
}

// PageableItem returns the field to be paginated over.
func (p *PaginationInfo) PageableItem() *Field {
	return p.pageableItem
}

// UpdateMethodPagination marks all methods that conform to
// [AIP-4233](https://google.aip.dev/client-libraries/4233) as pageable.
func UpdateMethodPagination(overrides []PaginationOverride, a *API) {
	for m := range a.AllMethods() {
		reqMsg := a.Message(m.InputTypeID)
		pageTokenField := paginationRequestInfo(reqMsg)
		if pageTokenField == nil {
			continue
		}

		respMsg := a.Message(m.OutputTypeID)
		paginationInfo := paginationResponseInfo(overrides, m.ID, respMsg)
		if paginationInfo == nil {
			continue
		}
		m.Pagination = pageTokenField
		respMsg.pagination = paginationInfo
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
	for _, field := range request.fields {
		// Some legacy services (e.g. sqladmin.googleapis.com)
		// predate AIP-4233 and use `maxResults` instead of
		// `pageSize` for the field name.
		if field.jsonName == pageSize && isPaginationPageSizeType(field) {
			return field
		}
		if field.jsonName == maxResults && isPaginationMaxResultsType(field) {
			return field
		}
	}
	return nil
}

func isPaginationPageSizeType(field *Field) bool {
	return field.typez == TypezInt32 || field.typez == TypezUint32
}

func isPaginationMaxResultsType(field *Field) bool {
	// Legacy maxResults types can be int32/uint32, and protobuf wrappers Int32Value/UInt32Value.
	if isPaginationPageSizeType(field) {
		return true
	}
	return field.typez == TypezMessage &&
		(field.typezID == ".google.protobuf.Int32Value" ||
			field.typezID == ".google.protobuf.UInt32Value")
}

func paginationRequestToken(request *Message) *Field {
	for _, field := range request.fields {
		if field.jsonName == pageToken && field.typez == TypezString {
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
		pageableItem:  pageableItem,
		nextPageToken: nextPageToken,
	}
}

func paginationResponseItem(overrides []PaginationOverride, methodID string, response *Message) *Field {
	idx := slices.IndexFunc(overrides, func(o PaginationOverride) bool { return o.ID == methodID })
	if idx != -1 {
		overrideName := overrides[idx].ItemField
		fieldIdx := slices.IndexFunc(response.fields, func(f *Field) bool { return f.name == overrideName })
		if fieldIdx == -1 {
			return nil
		}
		return response.fields[fieldIdx]
	}

	var mapItems *Field
	for _, field := range response.fields {
		if field.mapz && mapItems == nil {
			mapItems = field
		}
		if field.repeated && field.typez == TypezMessage {
			return field
		}
	}
	return mapItems
}

func paginationResponseNextPageToken(response *Message) *Field {
	for _, field := range response.fields {
		if field.jsonName == nextPageToken && field.typez == TypezString {
			return field
		}
	}
	return nil
}
