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

	"github.com/google/go-cmp/cmp"
)

func TestResolveAutoPopulatedFields(t *testing.T) {
	reqID := NewFieldBuilder().
		WithName("request_id").
		WithTypez(TypezString).
		WithAutoPopulated(true).
		Build()
	other := NewFieldBuilder().
		WithName("other").
		WithTypez(TypezString).
		Build()
	req := NewTestMessage("Request").
		WithPackage("test").
		WithFields(reqID, other)
	method := NewTestMethod("Create").
		WithID(".test.Service.Create").
		WithInput(req)
	svc := NewTestService("Service").
		WithPackage("test").
		WithMethods(method)
	model := NewTestAPI([]*Message{req}, nil, []*Service{svc})

	model.ResolveAutoPopulatedFields(map[string][]string{
		".test.Service.Create":  {"request_id", "other", "missing"},
		".test.Service.Unknown": {"request_id"},
	})

	want := []*Field{reqID}
	if diff := cmp.Diff(want, method.AutoPopulated(), allowUnexported()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
