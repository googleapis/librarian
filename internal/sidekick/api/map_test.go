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

func TestMap(t *testing.T) {
	key := NewFieldBuilder().WithName("key").WithTypez(TypezString).Build()
	val := NewFieldBuilder().WithName("value").WithTypez(TypezInt32).Build()
	got := NewMapBuilder().
		WithID(".test.MapEntry").
		WithKey(key).
		WithValue(val).
		Build()

	want := &Map{
		id:    ".test.MapEntry",
		key:   key,
		value: val,
	}
	opts := cmp.AllowUnexported(Map{}, Field{})
	if diff := cmp.Diff(want, got, opts); diff != "" {
		t.Errorf("Map mismatch (-want +got):\n%s", diff)
	}
	if got.ID() != want.id || got.Key() != key || got.Value() != val {
		t.Errorf("unexpected Map getters: %+v", got)
	}

	m2 := NewMapBuilderFrom(got).WithID(".test.MapEntry2").Build()
	if m2.ID() != ".test.MapEntry2" || m2.Key() != key || m2.Value() != val {
		t.Errorf("unexpected m2: %+v", m2)
	}
}
