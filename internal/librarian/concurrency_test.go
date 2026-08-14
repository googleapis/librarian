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

package librarian

import (
	"runtime"
	"testing"
)

func TestConcurrencyLimit(t *testing.T) {
	cpus := runtime.NumCPU()
	for _, test := range []struct {
		name string
		jobs int
		want int
	}{
		{name: "zero falls back to NumCPU", jobs: 0, want: cpus},
		{name: "negative falls back to NumCPU", jobs: -4, want: cpus},
		{name: "positive is used as-is", jobs: 3, want: 3},
		{name: "one", jobs: 1, want: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := concurrencyLimit(test.jobs); got != test.want {
				t.Errorf("concurrencyLimit(%d) = %d, want %d", test.jobs, got, test.want)
			}
		})
	}
}
