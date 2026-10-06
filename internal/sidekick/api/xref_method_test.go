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
	somePagination := NewTestField("page")
	someOperationInfo := NewTestOperationInfo("", "")
	someDiscoverLro := NewTestDiscoveryLro()
	for _, test := range []struct {
		name     string
		method   *Method
		isSimple bool
	}{
		{
			name:     "simple method",
			method:   NewTestMethod("test"),
			isSimple: true,
		},
		{
			name:     "pagination method",
			method:   NewTestMethod("test").WithPagination(somePagination),
			isSimple: false,
		},
		{
			name:     "client streaming method",
			method:   NewTestMethod("test").WithClientSideStreaming(),
			isSimple: false,
		},
		{
			name:     "server streaming method",
			method:   NewTestMethod("test").WithServerSideStreaming(),
			isSimple: false,
		},
		{
			name:     "LRO method",
			method:   NewTestMethod("test").WithOperationInfo(someOperationInfo),
			isSimple: false,
		},
		{
			name:     "Discovery LRO method",
			method:   NewTestMethod("test").WithDiscoveryLro(someDiscoverLro),
			isSimple: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			enrichMethodSamples(test.method)
			if got := test.method.IsSimple; got != test.isSimple {
				t.Errorf("IsSimple() = %v, want %v", got, test.isSimple)
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
			method: NewTestMethod("test"),
			want:   false,
		},
		{
			name:   "LRO method is LRO",
			method: NewTestMethod("test").WithOperationInfo(NewTestOperationInfo("", "")),
			want:   true,
		},
		{
			name:   "LRO method is discovery LRO",
			method: NewTestMethod("test").WithDiscoveryLro(NewTestDiscoveryLro()),
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
	responseMsg := NewTestMessage("Message").WithPackage("some.response")
	model := NewTestAPI([]*Message{responseMsg}, nil, nil)
	emptyMsg := model.Message(".google.protobuf.Empty")

	for _, test := range []struct {
		name         string
		method       *Method
		wantResponse *Message
		wantEmpty    bool
	}{
		{
			name: "LRO with empty response",
			method: func() *Method {
				m := NewTestMethod("test").WithOperationInfo(NewTestOperationInfo(emptyMsg.ID, ""))
				m.Model = model
				return m
			}(),
			wantResponse: emptyMsg,
			wantEmpty:    true,
		},
		{
			name: "LRO with non-empty response",
			method: func() *Method {
				m := NewTestMethod("test").WithOperationInfo(NewTestOperationInfo(responseMsg.ID, ""))
				m.Model = model
				return m
			}(),
			wantResponse: responseMsg,
			wantEmpty:    false,
		},
		{
			name: "non-LRO method",
			method: func() *Method {
				m := NewTestMethod("test")
				m.Model = model
				return m
			}(),
			wantResponse: nil,
			wantEmpty:    false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			enrichMethodSamples(test.method)
			if got := test.method.LongRunningResponseType; got != test.wantResponse {
				t.Errorf("LongRunningResponseType() = %v, want %v", got, test.wantResponse)
			}
			enrichMethodSamples(test.method)
			if got := test.method.LongRunningReturnsEmpty; got != test.wantEmpty {
				t.Errorf("LongRunningReturnsEmpty() = %v, want %v", got, test.wantEmpty)
			}
		})
	}
}

func TestIsList(t *testing.T) {
	paginationResp := NewTestMessage("Response").
		WithPagination(NewTestField("token"), NewTestField("item"))

	for _, test := range []struct {
		name   string
		method *Method
		want   bool
	}{
		{
			name:   "list method",
			method: NewTestMethod("test").WithOutput(paginationResp),
			want:   true,
		},
		{
			name:   "simple method",
			method: NewTestMethod("test"),
			want:   false,
		},
		{
			name:   "no output type",
			method: NewTestMethod("test").WithOutput(nil),
			want:   false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			enrichMethodSamples(test.method)
			if got := test.method.IsList; got != test.want {
				t.Errorf("IsList() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestIsStreaming(t *testing.T) {
	for _, test := range []struct {
		name   string
		method *Method
		want   bool
	}{
		{
			name:   "unary method",
			method: NewTestMethod("test"),
			want:   false,
		},
		{
			name:   "client streaming",
			method: NewTestMethod("test").WithClientSideStreaming(),
			want:   true,
		},
		{
			name:   "server streaming",
			method: NewTestMethod("test").WithServerSideStreaming(),
			want:   true,
		},
		{
			name:   "bidi streaming",
			method: NewTestMethod("test").WithBidiStreaming(),
			want:   true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			enrichMethodSamples(test.method)
			if got := test.method.IsStreaming; got != test.want {
				t.Errorf("IsStreaming() = %v, want %v", got, test.want)
			}
		})
	}
}
