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
			name:            "pre-1.0.0 feat is minor bump",
			highestChange:   Minor,
			currentVersion:  "0.2.3",
			expectedVersion: "0.3.0",
		},
		{
			name:            "pre-1.0.0 feat with downgrade is patch bump",
			highestChange:   Minor,
			currentVersion:  "0.2.3",
			expectedVersion: "0.2.4",
			opts:            DeriveNextOptions{DowngradePreGAChanges: true},
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
		{
			name:            "pre-1.0.0 prerelease with numeric trailer and bump core option",
			highestChange:   Minor,
			currentVersion:  "0.2.3-alpha.2",
			expectedVersion: "0.3.0-alpha.1",
			opts:            DeriveNextOptions{BumpVersionCore: true},
		},
		{
			name:            "pre-1.0.0 prerelease with numeric trailer, bump core and downgrade options",
			highestChange:   Minor,
			currentVersion:  "0.2.3-alpha.2",
			expectedVersion: "0.2.4-alpha.1",
			opts:            DeriveNextOptions{BumpVersionCore: true, DowngradePreGAChanges: true},
		},
		{
			name:            "pre-1.0.0 prerelease feat with downgrade, no bump core, is still prerelease bump",
			highestChange:   Minor,
			currentVersion:  "0.2.3-alpha.2",
			expectedVersion: "0.2.3-alpha.3",
			opts:            DeriveNextOptions{DowngradePreGAChanges: true},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			nextVersion, err := test.opts.DeriveNext(test.highestChange, test.currentVersion)
			if err != nil {
				t.Fatalf("DeriveNextOptions.DeriveNext() returned an error: %v", err)
			}
			if diff := cmp.Diff(test.expectedVersion, nextVersion); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDeriveNextOptions_DeriveNext_Error(t *testing.T) {
	for _, test := range []struct {
		name           string
		changeLevel    ChangeLevel
		currentVersion string
		wantErr        string
	}{
		{
			name:           "bad version",
			changeLevel:    Minor,
			currentVersion: "abc123",
			wantErr:        "failed to parse version",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := DeriveNextOptions{}.DeriveNext(test.changeLevel, test.currentVersion)
			if err == nil {
				t.Errorf("DeriveNextOptions.DeriveNext(%v, %q) did not return an error as expected.", test.changeLevel, test.currentVersion)
			} else if !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("mismatch, got %q, wanted inclusion of %q", err, test.wantErr)
			}
		})
	}
}

func TestDeriveNextOptions_DeriveNextPreview(t *testing.T) {
	for _, test := range []struct {
		name           string
		previewVersion string
		stableVersion  string
		opts           DeriveNextOptions
		want           string
	}{
		{
			name:           "equal, bump preview core, prerelease reset",
			previewVersion: "1.2.3-rc.3",
			stableVersion:  "1.2.3",
			want:           "1.3.0-rc.1",
		},
		{
			name:           "equal, pre-GA, bump preview core, prerelease reset",
			previewVersion: "0.1.2-rc.3",
			stableVersion:  "0.1.2",
			want:           "0.2.0-rc.1",
		},
		{
			name:           "equal, pre-GA, bump preview core, downgrade pre-GA, patch bump",
			previewVersion: "0.1.2-rc",
			stableVersion:  "0.1.2",
			want:           "0.1.3-rc",
			opts:           DeriveNextOptions{DowngradePreGAChanges: true},
		},
		{
			name:           "equal, bump preview core, no prerelease number",
			previewVersion: "1.2.3-rc",
			stableVersion:  "1.2.3",
			want:           "1.3.0-rc",
		},
		{
			name:           "stable ahead, catch up, bump preview core, prerelease reset",
			previewVersion: "1.2.3-rc.3",
			stableVersion:  "1.3.0",
			want:           "1.4.0-rc.1",
		},
		{
			name:           "stable ahead, pre-GA, catch up, bump preview core, prerelease reset",
			previewVersion: "0.2.3-rc.3",
			stableVersion:  "0.3.0",
			want:           "0.4.0-rc.1",
		},
		{
			name:           "stable ahead by major, catch up, bump preview core, prerelease reset",
			previewVersion: "1.2.3-rc.3",
			stableVersion:  "2.0.0",
			want:           "2.1.0-rc.1",
		},
		{
			name:           "stable ahead by major, pre-GA, catch up, minor bump, prerelease reset",
			previewVersion: "0.1.3-rc.3",
			stableVersion:  "1.0.0",
			want:           "1.1.0-rc.1",
		},
		{
			name:           "stable ahead, pre-GA, bump preview core, downgrade pre-GA, patch bump, no prerelease number",
			previewVersion: "0.1.2-rc",
			stableVersion:  "0.1.3",
			want:           "0.1.4-rc",
			opts:           DeriveNextOptions{DowngradePreGAChanges: true},
		},
		{
			name:           "preview ahead, bump prerelease number",
			previewVersion: "1.2.4-rc.1",
			stableVersion:  "1.2.3",
			want:           "1.2.4-rc.2",
		},
		{
			name:           "preview ahead, pre-GA, bump prerelease number",
			previewVersion: "0.1.3-rc.1",
			stableVersion:  "0.1.2",
			want:           "0.1.3-rc.2",
		},
		{
			name:           "preview ahead, no prerelease number, append prerelease number",
			previewVersion: "1.2.4-rc",
			stableVersion:  "1.2.3",
			want:           "1.2.4-rc.1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			nextVersion, err := test.opts.DeriveNextPreview(test.previewVersion, test.stableVersion)
			if err != nil {
				t.Fatalf("DeriveNextOptions.DeriveNextPreview() returned an error: %v", err)
			}
			if diff := cmp.Diff(test.want, nextVersion); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDeriveNextOptions_DeriveNextPreview_Errors(t *testing.T) {
	for _, test := range []struct {
		name           string
		previewVersion string
		stableVersion  string
		wantErr        string
	}{
		{
			name:           "bad preview version",
			previewVersion: "abc123",
			stableVersion:  "1.2.3",
			wantErr:        "parse preview version",
		},
		{
			name:           "bad stable version",
			previewVersion: "0.1.2-rc.3",
			stableVersion:  "abc123",
			wantErr:        "parse stable version",
		},
		{
			name:           "non-prerelease preview version",
			previewVersion: "0.1.3",
			stableVersion:  "0.1.2",
			wantErr:        "no prerelease segment",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := DeriveNextOptions{}.DeriveNextPreview(test.previewVersion, test.stableVersion)
			if err == nil {
				t.Errorf("DeriveNextOptions.DeriveNextPreview(%q, %q) did not return an error as expected.", test.previewVersion, test.stableVersion)
			} else if !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("mismatch, got %q, wanted inclusion of %q", err, test.wantErr)
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
