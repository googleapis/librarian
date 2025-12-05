// Copyright 2025 Google LLC
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

package semver

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func ptr(i int) *int {
	return &i
}

func TestParse(t *testing.T) {
	for _, test := range []struct {
		name          string
		version       string
		want          version
		wantErr       bool
		wantErrPhrase string
	}{
		{
			name:    "valid version",
			version: "1.2.3",
			want: version{
				Major: 1,
				Minor: 2,
				Patch: 3,
			},
		},
		{
			name:          "invalid version with v prefix",
			version:       "v1.2.3",
			wantErr:       true,
			wantErrPhrase: "invalid version format",
		},
		{
			name:    "valid version with prerelease",
			version: "1.2.3-alpha.1",
			want: version{
				Major:               1,
				Minor:               2,
				Patch:               3,
				Prerelease:          "alpha",
				PrereleaseSeparator: ".",
				PrereleaseNumber:    ptr(1),
			},
		},
		{
			name:    "valid version with format 1.2.3-betaXX",
			version: "1.2.3-beta21",
			want: version{
				Major:            1,
				Minor:            2,
				Patch:            3,
				Prerelease:       "beta",
				PrereleaseNumber: ptr(21),
			},
		},
		{
			name:    "valid version with prerelease without version",
			version: "1.2.3-beta",
			want: version{
				Major:      1,
				Minor:      2,
				Patch:      3,
				Prerelease: "beta",
			},
		},
		{
			name:    "valid shortened version",
			version: "1.2",
			want: version{
				Major: 1,
				Minor: 2,
				Patch: 0,
			},
		},
		{
			name:          "invalid prerelease number with separator",
			version:       "1.2.3-rc.abc",
			wantErr:       true,
			wantErrPhrase: "invalid prerelease number",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			actual, err := parse(test.version)
			if test.wantErr {
				if err == nil {
					t.Fatal("Parse() should have failed")
				}
				if !strings.Contains(err.Error(), test.wantErrPhrase) {
					t.Errorf("Parse() returned error %q, want to contain %q", err.Error(), test.wantErrPhrase)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse() failed: %v", err)
			}
			if diff := cmp.Diff(test.want, actual); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestVersion_String(t *testing.T) {
	for _, test := range []struct {
		name     string
		version  version
		expected string
	}{
		{
			name: "simple version",
			version: version{
				Major: 1,
				Minor: 2,
				Patch: 3,
			},
			expected: "1.2.3",
		},
		{
			name: "with prerelease",
			version: version{
				Major:               1,
				Minor:               2,
				Patch:               3,
				Prerelease:          "alpha",
				PrereleaseSeparator: ".",
				PrereleaseNumber:    ptr(1),
			},
			expected: "1.2.3-alpha.1",
		},
		{
			name: "with prerelease no separator",
			version: version{
				Major:            1,
				Minor:            2,
				Patch:            3,
				Prerelease:       "beta",
				PrereleaseNumber: ptr(21),
			},
			expected: "1.2.3-beta21",
		},
		{
			name: "with prerelease no version",
			version: version{
				Major:      1,
				Minor:      2,
				Patch:      3,
				Prerelease: "beta",
			},
			expected: "1.2.3-beta",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if actual := test.version.String(); actual != test.expected {
				t.Errorf("String() = %q, want %q", actual, test.expected)
			}
		})
	}
}

func TestDeriveNext(t *testing.T) {
	for _, test := range []struct {
		name            string
		highestChange   ChangeLevel
		currentVersion  string
		expectedVersion string
	}{
		{
			name:            "major bump",
			highestChange:   Major,
			currentVersion:  "1.2.3",
			expectedVersion: "2.0.0",
		},
		{
			name:            "minor bump",
			highestChange:   Minor,
			currentVersion:  "1.2.3",
			expectedVersion: "1.3.0",
		},
		{
			name:            "patch bump",
			highestChange:   Patch,
			currentVersion:  "1.2.3",
			expectedVersion: "1.2.4",
		},
		{
			name:            "pre-1.0.0 feat is patch bump",
			highestChange:   Minor, // feat is minor
			currentVersion:  "0.2.3",
			expectedVersion: "0.3.0",
		},
		{
			name:            "pre-1.0.0 fix is patch bump",
			highestChange:   Patch,
			currentVersion:  "0.2.3",
			expectedVersion: "0.2.4",
		},
		{
			name:            "pre-1.0.0 breaking change is minor bump",
			highestChange:   Major,
			currentVersion:  "0.2.3",
			expectedVersion: "0.3.0",
		},
		{
			name:            "prerelease bump with numeric trailer",
			highestChange:   Minor,
			currentVersion:  "1.2.3-beta.1",
			expectedVersion: "1.2.3-beta.2",
		},
		{
			name:            "prerelease bump without numeric trailer",
			highestChange:   Patch,
			currentVersion:  "1.2.3-beta",
			expectedVersion: "1.2.3-beta.1",
		},
		{
			name:            "prerelease bump with betaXX format",
			highestChange:   Major,
			currentVersion:  "1.2.3-beta21",
			expectedVersion: "1.2.3-beta22",
		},
		{
			name:            "no bump",
			highestChange:   None,
			currentVersion:  "1.2.3",
			expectedVersion: "1.2.3",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			nextVersion, err := DeriveNext(test.highestChange, test.currentVersion)
			if err != nil {
				t.Fatalf("DeriveNext() returned an error: %v", err)
			}
			if diff := cmp.Diff(test.expectedVersion, nextVersion); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDeriveNextOptions_DeriveNext(t *testing.T) {
	for _, test := range []struct {
		name            string
		highestChange   ChangeLevel
		currentVersion  string
		expectedVersion string
		opts            DeriveNextOptions
	}{
		{
			name:            "major bump",
			highestChange:   Major,
			currentVersion:  "1.2.3",
			expectedVersion: "2.0.0",
		},
		{
			name:            "minor bump",
			highestChange:   Minor,
			currentVersion:  "1.2.3",
			expectedVersion: "1.3.0",
		},
		{
			name:            "patch bump",
			highestChange:   Patch,
			currentVersion:  "1.2.3",
			expectedVersion: "1.2.4",
		},
		{
			name:            "pre-1.0.0 feat is patch bump",
			highestChange:   Minor, // feat is minor
			currentVersion:  "0.2.3",
			expectedVersion: "0.3.0",
		},
		{
			name:            "pre-1.0.0 fix is patch bump",
			highestChange:   Patch,
			currentVersion:  "0.2.3",
			expectedVersion: "0.2.4",
		},
		{
			name:            "pre-1.0.0 breaking change is minor bump",
			highestChange:   Major,
			currentVersion:  "0.2.3",
			expectedVersion: "0.3.0",
		},
		{
			name:            "prerelease bump with numeric trailer",
			highestChange:   Minor,
			currentVersion:  "1.2.3-beta.1",
			expectedVersion: "1.2.3-beta.2",
		},
		{
			name:            "prerelease bump without numeric trailer",
			highestChange:   Patch,
			currentVersion:  "1.2.3-beta",
			expectedVersion: "1.2.3-beta.1",
		},
		{
			name:            "prerelease bump with betaXX format",
			highestChange:   Major,
			currentVersion:  "1.2.3-beta21",
			expectedVersion: "1.2.3-beta22",
		},
		{
			name:            "no bump",
			highestChange:   None,
			currentVersion:  "1.2.3",
			expectedVersion: "1.2.3",
		},
		{
			name:            "prerelease with bump core option",
			highestChange:   Minor,
			currentVersion:  "1.2.3-alpha",
			expectedVersion: "1.3.0-alpha",
			opts:            DeriveNextOptions{BumpVersionCore: true},
		},
		{
			name:            "prerelease with numeric trailer and bump core option",
			highestChange:   Minor,
			currentVersion:  "1.2.3-alpha.5",
			expectedVersion: "1.3.0-alpha.1",
			opts:            DeriveNextOptions{BumpVersionCore: true},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			nextVersion, err := test.opts.DeriveNext(test.highestChange, test.currentVersion)
			if err != nil {
				t.Fatalf("DeriveNext() returned an error: %v", err)
			}
			if diff := cmp.Diff(test.expectedVersion, nextVersion); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMaxVersion(t *testing.T) {
	for _, test := range []struct {
		name     string
		versions []string
		want     string
	}{
		{
			name:     "empty",
			versions: []string{},
			want:     "",
		},
		{
			name:     "single",
			versions: []string{"1.2.3"},
			want:     "1.2.3",
		},
		{
			name:     "multiple",
			versions: []string{"1.2.3", "1.2.4", "1.2.2"},
			want:     "1.2.4",
		},
		{
			name:     "multiple with pre-release",
			versions: []string{"1.2.4", "1.2.4-alpha", "1.2.4-beta"},
			want:     "1.2.4",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := MaxVersion(test.versions...)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("TestMaxVersion() returned diff (-want +got):\n%s", diff)
			}
		})
	}
}
