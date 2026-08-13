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

import "testing"

func TestHasMessages(t *testing.T) {
	m := &Message{
		Name:    "Message",
		Package: "test",
		ID:      ".test.Message",
	}
	model := NewTestAPI([]*Message{m}, []*Enum{}, []*Service{})

	if !model.HasMessages() {
		t.Errorf("expected HasMessages() == true for: %v", model)
	}

	e := &Enum{
		Name:    "Enum",
		Package: "test",
		ID:      ".test.Enum",
	}
	model = NewTestAPI([]*Message{}, []*Enum{e}, []*Service{})
	if model.HasMessages() {
		t.Errorf("expected HasMessages() == false for: %v", model)
	}
}

func TestMethodIsBidiStreaming(t *testing.T) {
	for _, tc := range []struct {
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
			name:   "client streaming only",
			method: &Method{ClientSideStreaming: true},
			want:   false,
		},
		{
			name:   "server streaming only",
			method: &Method{ServerSideStreaming: true},
			want:   false,
		},
		{
			name:   "bidi streaming",
			method: &Method{ClientSideStreaming: true, ServerSideStreaming: true},
			want:   true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.method.IsBidiStreaming(); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
