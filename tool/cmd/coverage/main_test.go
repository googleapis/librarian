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

package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestTargetFor(t *testing.T) {
	// Verify each entry in the targets map is returned for its suffix.
	for suffix, want := range targets {
		got := targetFor([]string{"./" + suffix})
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("%s mismatch (-want +got):\n%s", suffix, diff)
		}
		// Also matches with /... wildcard.
		got = targetFor([]string{"./" + suffix + "/..."})
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("%s/... mismatch (-want +got):\n%s", suffix, diff)
		}
	}
	// Verify an unrecognized package returns defaultTarget.
	got := targetFor([]string{"./internal/something/else"})
	if diff := cmp.Diff(float64(defaultTarget), got); diff != "" {
		t.Errorf("default mismatch (-want +got):\n%s", diff)
	}
}

func TestParseTotalCoverage(t *testing.T) {
	for _, test := range []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{
			name: "typical output",
			input: `github.com/foo/bar/baz.go:10:	Foo		100.0%
github.com/foo/bar/baz.go:20:	Bar		50.0%
total:					(statements)	75.0%
`,
			want: 75.0,
		},
		{
			name: "no trailing newline",
			input: `github.com/foo/bar/baz.go:10:	Foo		100.0%
total:					(statements)	89.6%`,
			want: 89.6,
		},
		{
			name:    "no total line",
			input:   "github.com/foo/bar/baz.go:10:\tFoo\t\t100.0%\n",
			wantErr: true,
		},
		{
			name:    "empty output",
			input:   "",
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseTotalCoverage(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
