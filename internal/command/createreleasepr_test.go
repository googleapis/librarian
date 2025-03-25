// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package command

import (
	"testing"

	"github.com/googleapis/librarian/internal/statepb"
)

func TestCalculateNextVersion(t *testing.T) {
	tests := []struct {
		current string
		want    string // Empty for an error
	}{
		{
			current: "1.0.0",
			want:    "1.1.0",
		},
		{
			current: "1.2.3-beta01",
			want:    "1.2.3-beta02",
		},
		{
			current: "1.2.3-alpha99",
			want:    "1.2.3-alpha100",
		},
		{
			current: "1.2.3-beta.1",
			want:    "1.2.3-beta.2",
		},
		{
			current: "1.2.3-beta.9",
			want:    "1.2.3-beta.10",
		},
		{
			current: "1.2.3-bad",
			want:    "",
		},
		{
			current: "bad",
			want:    "",
		},
	}
	for _, test := range tests {
		library := &statepb.LibraryState{
			CurrentVersion: test.current,
		}
		got, err := calculateNextVersion(library)
		if test.want == "" {
			if err == nil {
				t.Errorf("calculateNextVersion(%s); error expected", test.current)
			}
			continue
		}
		if err != nil {
			t.Errorf("calculateNextVersion(%s); got error %v", test.current, err)
			continue
		}

		if test.want != got {
			t.Errorf("calculateNextVersion(%s) expected %s, got %s", test.current, test.want, got)
		}
	}
}
