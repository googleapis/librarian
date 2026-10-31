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
	"os"
	"path/filepath"
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

func TestPackageCoverage(t *testing.T) {
	profile := `mode: atomic
example.com/a/x.go:1.1,2.2 3 1
example.com/a/y.go:1.1,2.2 1 0
example.com/b/z.go:1.1,2.2 2 0
example.com/b/z.go:3.1,4.2 2 5
`
	got, err := packageCoverage(profile)
	if err != nil {
		t.Fatal(err)
	}
	want := []pkgCoverage{
		{pkg: "example.com/a", covered: 3, total: 4},
		{pkg: "example.com/b", covered: 2, total: 4},
	}
	if diff := cmp.Diff(want, got, cmp.AllowUnexported(pkgCoverage{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestPackageCoverage_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		profile string
	}{
		{"wrong field count", "example.com/a/x.go:1.1,2.2 3\n"},
		{"missing colon", "example.com/a/x.go 3 1\n"},
		{"bad statements", "example.com/a/x.go:1.1,2.2 x 1\n"},
		{"bad count", "example.com/a/x.go:1.1,2.2 3 x\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := packageCoverage(test.profile); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestFormatSummary(t *testing.T) {
	pkgs := []pkgCoverage{
		{pkg: "example.com/a", covered: 3, total: 4},
		{pkg: "example.com/empty"},
	}
	for _, test := range []struct {
		name  string
		total float64
		want  string
	}{
		{
			name:  "pass",
			total: 85,
			want: "### Coverage: 85.0% (target: 80%) ✅ PASS\n\n" +
				"<details><summary>Per-package statement coverage</summary>\n\n" +
				"| Package | Coverage | Statements |\n|---|---:|---:|\n" +
				"| `example.com/a` | 75.0% | 3/4 |\n" +
				"| `example.com/empty` | 0.0% | 0/0 |\n" +
				"\n</details>\n\n",
		},
		{
			name:  "fail",
			total: 79.9,
			want: "### Coverage: 79.9% (target: 80%) ❌ FAIL\n\n" +
				"<details><summary>Per-package statement coverage</summary>\n\n" +
				"| Package | Coverage | Statements |\n|---|---:|---:|\n" +
				"| `example.com/a` | 75.0% | 3/4 |\n" +
				"| `example.com/empty` | 0.0% | 0/0 |\n" +
				"\n</details>\n\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := formatSummary(test.total, 80, pkgs)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestWriteStepSummary(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("coverage.out", []byte("mode: atomic\nexample.com/a/x.go:1.1,2.2 1 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	summary := filepath.Join(t.TempDir(), "summary.md")
	t.Setenv("GITHUB_STEP_SUMMARY", summary)
	if err := writeStepSummary(100, 80); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(summary)
	if err != nil {
		t.Fatal(err)
	}
	want := formatSummary(100, 80, []pkgCoverage{{pkg: "example.com/a", covered: 1, total: 1}})
	if diff := cmp.Diff(want, string(got)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestWriteStepSummary_Unset(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GITHUB_STEP_SUMMARY", "")
	// No coverage.out exists, so any attempt to read it would fail.
	if err := writeStepSummary(100, 80); err != nil {
		t.Fatal(err)
	}
}
