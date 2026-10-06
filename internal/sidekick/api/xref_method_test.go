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
)

func TestIsSimpleMethod(t *testing.T) {
	somePagination := &Field{}
	someOperationInfo := &OperationInfo{}
	someDiscoverLro := &DiscoveryLro{}
	testCases := []struct {
		name     string
		method   *Method
		isSimple bool
	}{
		{
			name:     "simple method",
			method:   &Method{},
			isSimple: true,
		},
		{
			name:     "pagination method",
			method:   &Method{Pagination: somePagination},
			isSimple: false,
		},
		{
			name:     "client streaming method",
			method:   &Method{ClientSideStreaming: true},
			isSimple: false,
		},
		{
			name:     "server streaming method",
			method:   &Method{ServerSideStreaming: true},
			isSimple: false,
		},
		{
			name:     "LRO method",
			method:   &Method{OperationInfo: someOperationInfo},
			isSimple: false,
		},
		{
			name:     "Discovery LRO method",
			method:   &Method{DiscoveryLro: someDiscoverLro},
			isSimple: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			enrichMethodSamples(tc.method)
			if got := tc.method.IsSimple; got != tc.isSimple {
				t.Errorf("IsSimple() = %v, want %v", got, tc.isSimple)
			}
		})
	}
}

func TestIsLRO(t *testing.T) {
	for _, test := range []struct {
		name   string
		method *Method
		want   bool
	}{
		{
			name:   "simple method is not LRO",
			method: &Method{},
			want:   false,
		},
		{
			name:   "LRO method is LRO",
			method: &Method{OperationInfo: &OperationInfo{}},
			want:   true,
		},
		{
			name:   "LRO method is discovery LRO",
			method: &Method{DiscoveryLro: &DiscoveryLro{}},
			want:   true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			enrichMethodSamples(test.method)
			if got := test.method.IsLRO; got != test.want {
				t.Errorf("IsLRO() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestLongRunningHelpers(t *testing.T) {
	emptyMsg := &Message{ID: ".google.protobuf.Empty"}
	responseMsg := &Message{ID: "some.response.Message"}
	model := &API{
		messageByID: map[string]*Message{
			emptyMsg.ID:    emptyMsg,
			responseMsg.ID: responseMsg,
		},
	}

	testCases := []struct {
		name         string
		method       *Method
		wantResponse *Message
		wantEmpty    bool
	}{
		{
			name: "LRO with empty response",
			method: &Method{
				OperationInfo: &OperationInfo{ResponseTypeID: emptyMsg.ID},
				Model:         model,
			},
			wantResponse: emptyMsg,
			wantEmpty:    true,
		},
		{
			name: "LRO with non-empty response",
			method: &Method{
				OperationInfo: &OperationInfo{ResponseTypeID: responseMsg.ID},
				Model:         model,
			},
			wantResponse: responseMsg,
			wantEmpty:    false,
		},
		{
			name: "non-LRO method",
			method: &Method{
				Model: model,
			},
			wantResponse: nil,
			wantEmpty:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			enrichMethodSamples(tc.method)
			if got := tc.method.LongRunningResponseType; got != tc.wantResponse {
				t.Errorf("LongRunningResponseType() = %v, want %v", got, tc.wantResponse)
			}
			enrichMethodSamples(tc.method)
			if got := tc.method.LongRunningReturnsEmpty; got != tc.wantEmpty {
				t.Errorf("LongRunningReturnsEmpty() = %v, want %v", got, tc.wantEmpty)
			}
		})
	}
}

func TestIsList(t *testing.T) {
	testCases := []struct {
		name   string
		method *Method
		want   bool
	}{
		{
			name:   "list method",
			method: &Method{OutputType: &Message{Pagination: &PaginationInfo{}}},
			want:   true,
		},
		{
			name:   "simple method",
			method: &Method{},
			want:   false,
		},
		{
			name:   "no output type",
			method: &Method{OutputType: nil},
			want:   false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			enrichMethodSamples(tc.method)
			if got := tc.method.IsList; got != tc.want {
				t.Errorf("IsList() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsStreaming(t *testing.T) {
	testCases := []struct {
		name   string
		method *Method
		want   bool
	}{
		{
			name:   "unary method",
			method: &Method{},
			want:   false,
		},
		{
			name:   "client streaming",
			method: &Method{ClientSideStreaming: true},
			want:   true,
		},
		{
			name:   "server streaming",
			method: &Method{ServerSideStreaming: true},
			want:   true,
		},
		{
			name:   "bidi streaming",
			method: &Method{ClientSideStreaming: true, ServerSideStreaming: true},
			want:   true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			enrichMethodSamples(tc.method)
			if got := tc.method.IsStreaming; got != tc.want {
				t.Errorf("IsStreaming() = %v, want %v", got, tc.want)
			}
		})
	}
}
