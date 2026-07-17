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

func TestUpdateMethodIdempotency(t *testing.T) {
	api := &API{}
	m1 := &Method{ID: "m1"}
	m2 := &Method{ID: "m2"}
	api.AddMethod(m1)
	api.AddMethod(m2)

	overrides := []IdempotencyOverride{
		{ID: "m1", IsIdempotent: true},
	}
	UpdateMethodIdempotency(overrides, api)

	if m1.IdempotencyOverride == nil || !*m1.IdempotencyOverride {
		t.Errorf("expected m1.IdempotencyOverride to be true, got %v", m1.IdempotencyOverride)
	}
	if m2.IdempotencyOverride != nil {
		t.Errorf("expected m2.IdempotencyOverride to be nil, got %v", m2.IdempotencyOverride)
	}
}
