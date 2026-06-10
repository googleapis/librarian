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
	"log/slog"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/command"
)

func TestVerboseFlag(t *testing.T) {
	oldDefault := slog.Default()
	t.Cleanup(func() {
		command.Verbose = false
		slog.SetDefault(oldDefault)
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
			if diff := cmp.Diff(test.wantVerbose, command.Verbose); diff != "" {
				t.Errorf("command.Verbose mismatch (-want +got):\n%s", diff)
			}

			// Verify slog level configuration.
			ctx := t.Context()
			if diff := cmp.Diff(test.wantVerbose, slog.Default().Enabled(ctx, slog.LevelDebug)); diff != "" {
				t.Errorf("slog.Default().Enabled(Debug) mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, slog.Default().Enabled(ctx, slog.LevelWarn)); diff != "" {
				t.Errorf("slog.Default().Enabled(Warn) mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
