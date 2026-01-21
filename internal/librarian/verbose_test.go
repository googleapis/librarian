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
	"testing"

	"github.com/googleapis/librarian/internal/command"
)

func TestVerboseFlag(t *testing.T) {
	t.Cleanup(func() {
		command.Verbose = false
	})

	for _, test := range []struct {
		name        string
		args        []string
		wantVerbose bool
	}{
		{"without verbose flag", []string{"librarian", "version"}, false},
		{"with -v flag", []string{"librarian", "-v", "version"}, true},
		{"with --verbose flag", []string{"librarian", "--verbose", "version"}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			command.Verbose = false
			if err := Run(t.Context(), test.args...); err != nil {
				t.Fatal(err)
			}
			if command.Verbose != test.wantVerbose {
				t.Errorf("command.Verbose = %t, want %t", command.Verbose, test.wantVerbose)
			}
		})
	}
}
