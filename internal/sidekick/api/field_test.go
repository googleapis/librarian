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

func TestWithNumber(t *testing.T) {
	for _, test := range []struct {
		name   string
		number int32
	}{
		{"tag 1", 1},
		{"tag 42", 42},
		{"large tag number", 536870911},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := NewTestField("test_field")
			ret := f.WithNumber(test.number)
			if ret != f {
				t.Errorf("Field.WithNumber(%d) did not return receiver pointer", test.number)
			}
			if got := f.Number; got != test.number {
				t.Errorf("Field.WithNumber(%d).Number = %d, want %d", test.number, got, test.number)
			}
		})
	}
}
