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
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func ptr(i int) *int {
	return &i
}

func TestParse(t *testing.T) {
	for _, test := range []struct {
		name    string
		version string
		want    version
	}{
		{
			name:    "valid version",
			version: "1.2.3",
			want: version{
				Major:       1,
				Minor:       2,
				Patch:       3,
				SpecVersion: SemVerSpecV2,
			},
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
				SpecVersion:         SemVerSpecV2,
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
				SpecVersion:      SemVerSpecV1,
			},
		},
		{
			name:    "valid version with prerelease without version",
			version: "1.2.3-beta",
			want: version{
				Major:       1,
				Minor:       2,
				Patch:       3,
				Prerelease:  "beta",
				SpecVersion: SemVerSpecV2,
			},
		},
		{
			name:    "valid shortened version",
			version: "1.2",
			want: version{
				Major:       1,
				Minor:       2,
				Patch:       0,
				SpecVersion: SemVerSpecV2,
			},
		},
		{
			name:    "valid version with format 1.2.3-alpha<digits>",
			version: "1.2.3-alpha1",
			want: version{
				Major:            1,
				Minor:            2,
				Patch:            3,
				Prerelease:       "alpha",
				PrereleaseNumber: ptr(1),
				SpecVersion:      SemVerSpecV1,
			},
		},
		{
			name:    "valid version with format 1.2.3-beta<digits>",
			version: "1.2.3-beta2",
			want: version{
				Major:            1,
				Minor:            2,
				Patch:            3,
				Prerelease:       "beta",
				PrereleaseNumber: ptr(2),
				SpecVersion:      SemVerSpecV1,
			},
		},
		{
			name:    "valid version with format 1.2.3-rc<digits>",
			version: "1.2.3-rc3",
			want: version{
				Major:            1,
				Minor:            2,
				Patch:            3,
				Prerelease:       "rc",
				PrereleaseNumber: ptr(3),
				SpecVersion:      SemVerSpecV1,
			},
		},
		{
			name:    "valid version with format 1.2.3-preview<digits>",
			version: "1.2.3-preview4",
			want: version{
				Major:            1,
				Minor:            2,
				Patch:            3,
				Prerelease:       "preview",
				PrereleaseNumber: ptr(4),
				SpecVersion:      SemVerSpecV1,
			},
		},
		{
			name:    "valid version with format 1.2.3-a<digits>",
			version: "1.2.3-a5",
			want: version{
				Major:            1,
				Minor:            2,
				Patch:            3,
				Prerelease:       "a",
				PrereleaseNumber: ptr(5),
				SpecVersion:      SemVerSpecV1,
			},
		},
		{
			name:    "valid version with format 1.2.3-b<digits>",
			version: "1.2.3-b6",
			want: version{
				Major:            1,
				Minor:            2,
				Patch:            3,
				Prerelease:       "b",
				PrereleaseNumber: ptr(6),
				SpecVersion:      SemVerSpecV1,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			actual, err := parse(test.version)
			if err != nil {
				t.Fatalf("Parse() failed: %v", err)
			}
			if diff := cmp.Diff(test.want, actual); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParse_Errors(t *testing.T) {
	for _, test := range []struct {
		name    string
		version string
		wantErr error
	}{
		{
			name:    "invalid version with v prefix",
			version: "v1.2.3",
			wantErr: errInvalidVersion,
		},
		{
			name:    "invalid prerelease number with separator",
			version: "1.2.3-rc.abc",
			wantErr: errInvalidPrereleaseNumber,
		},
		{
			name:    "invalid major number",
			version: "a.2.3",
			wantErr: errInvalidVersion,
		},
		{
			name:    "invalid minor number",
			version: "1.a.3",
			wantErr: errInvalidVersion,
		},
		{
			name:    "invalid patch number",
			version: "1.2.a",
			wantErr: errInvalidVersion,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, gotErr := parse(test.version)
			if gotErr == nil {
				t.Errorf("Parse(%q) should have failed", test.version)
			} else if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("Parse(%q) returned error %v, wanted %v", test.version, gotErr, test.wantErr)
			}
		})
	}
}

func TestParse_Invalid(t *testing.T) {
	for _, version := range []string{
		"1.2.3-0123",
		"1.2.3-0123.0123",
		"1.1.2+.123",
		"+invalid",
		"-invalid",
		"-invalid+invalid",
		"-invalid.01",
		"alpha",
		"alpha.beta",
		"alpha.beta.1",
		"alpha.1",
		"alpha+beta",
		"alpha_beta",
		"alpha.",
		"alpha..",
		"beta",
		"1.0.0-alpha_beta",
		"-alpha.",
		"1.0.0-alpha..",
		"1.0.0-alpha..1",
		"1.0.0-alpha...1",
		"1.0.0-alpha....1",
		"1.0.0-alpha.....1",
		"1.0.0-alpha......1",
		"1.0.0-alpha.......1",
		"01.1.1",
		"1.01.1",
		"1.1.01",
		"1.2.3.DEV",
		"1.2-SNAPSHOT",
		"1.2.31.2.3----RC-SNAPSHOT.12.09.1--..12+788",
		"1.2-RC-SNAPSHOT",
		"-1.0.3-gamma+b7718",
		"+justmeta",
		"9.8.7+meta+meta",
		"9.8.7-whatever+meta+meta",
		"99999999999999999999999.999999999999999999.99999999999999999----RC-SNAPSHOT.12.09.1--------------------------------..12",
	} {
		t.Run(version, func(t *testing.T) {
			if _, err := parse(version); err == nil {
				t.Error("Parse() should have failed")
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
			name: "with prerelease, semver spec v1 no separator",
			version: version{
				Major:            1,
				Minor:            2,
				Patch:            3,
				Prerelease:       "beta",
				PrereleaseNumber: ptr(21),
				SpecVersion:      SemVerSpecV1,
			},
			expected: "1.2.3-beta21",
		},
		{
			name: "with prerelease, semver spec v1 no separator, zero padded single digit",
			version: version{
				Major:            1,
				Minor:            2,
				Patch:            3,
				Prerelease:       "beta",
				PrereleaseNumber: ptr(2),
				SpecVersion:      SemVerSpecV1,
			},
			expected: "1.2.3-beta02",
		},
		{
			name: "with prerelease no number",
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
		name           string
		highestChange  ChangeLevel
		currentVersion string
		want           string
	}{
		{
			name:           "major bump",
			highestChange:  Major,
			currentVersion: "1.2.3",
			want:           "2.0.0",
		},
		{
			name:           "minor bump",
			highestChange:  Minor,
			currentVersion: "1.2.3",
			want:           "1.3.0",
		},
		{
			name:           "patch bump",
			highestChange:  Patch,
			currentVersion: "1.2.3",
			want:           "1.2.4",
		},
		{
			name:           "pre-1.0.0 feat is patch bump",
			highestChange:  Minor, // feat is minor
			currentVersion: "0.2.3",
			want:           "0.3.0",
		},
		{
			name:           "pre-1.0.0 fix is patch bump",
			highestChange:  Patch,
			currentVersion: "0.2.3",
			want:           "0.2.4",
		},
		{
			name:           "pre-1.0.0 breaking change is minor bump",
			highestChange:  Major,
			currentVersion: "0.2.3",
			want:           "0.3.0",
		},
		{
			name:           "prerelease bump with numeric trailer",
			highestChange:  Minor,
			currentVersion: "1.2.3-beta.1",
			want:           "1.2.3-beta.2",
		},
		{
			name:           "prerelease bump without numeric trailer",
			highestChange:  Patch,
			currentVersion: "1.2.3-beta",
			want:           "1.2.3-beta.1",
		},
		{
			name:           "prerelease bump with betaXX format",
			highestChange:  Major,
			currentVersion: "1.2.3-beta21",
			want:           "1.2.3-beta22",
		},
		{
			name:           "no bump",
			highestChange:  None,
			currentVersion: "1.2.3",
			want:           "1.2.3",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := DeriveNext(test.highestChange, test.currentVersion, DeriveNextOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Errorf("DeriveNext(%v, %q) = %q, want %q", test.highestChange, test.currentVersion, got, test.want)
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
			nextVersion, err := DeriveNext(test.highestChange, test.currentVersion, test.opts)
			if err != nil {
				t.Fatal(err)
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
		wantErr        error
	}{
		{
			name:           "bad version",
			changeLevel:    Minor,
			currentVersion: "abc123",
			wantErr:        errInvalidVersion,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := DeriveNext(test.changeLevel, test.currentVersion, DeriveNextOptions{})
			if err == nil {
				t.Errorf("DeriveNextOptions.DeriveNext(%v, %q) did not return an error as expected.", test.changeLevel, test.currentVersion)
			} else if !errors.Is(err, test.wantErr) {
				t.Errorf("DeriveNextOptions.DeriveNext(%v, %q), returned error %v, wanted %v", test.changeLevel, test.currentVersion, err, test.wantErr)
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
			name:           "preview ahead, ignore BumpVersionCore, bump prerelease number",
			previewVersion: "1.2.4-rc.1",
			stableVersion:  "1.2.3",
			want:           "1.2.4-rc.2",
			opts:           DeriveNextOptions{BumpVersionCore: true},
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
		{
			name:           "preview ahead, pre-GA, ignore BumpVersionCore, downgrade pre-GA, append prerelease number",
			previewVersion: "0.1.2-rc",
			stableVersion:  "0.1.1",
			want:           "0.1.2-rc.1",
			opts:           DeriveNextOptions{BumpVersionCore: true, DowngradePreGAChanges: true},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			nextVersion, err := DeriveNextPreview(test.previewVersion, test.stableVersion, test.opts)
			if err != nil {
				t.Fatalf("DeriveNextPreview() returned an error: %v", err)
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
		wantErr        error
	}{
		{
			name:           "bad preview version",
			previewVersion: "abc123",
			stableVersion:  "1.2.3",
			wantErr:        errInvalidPreviewVersion,
		},
		{
			name:           "bad stable version",
			previewVersion: "0.1.2-rc.3",
			stableVersion:  "abc123",
			wantErr:        errInvalidStableVersion,
		},
		{
			name:           "non-prerelease preview version",
			previewVersion: "0.1.3",
			stableVersion:  "0.1.2",
			wantErr:        errPreviewMissingPrerelease,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := DeriveNextPreview(test.previewVersion, test.stableVersion, DeriveNextOptions{})
			if err == nil {
				t.Errorf("DeriveNextPreview(%q, %q) did not return an error as expected.", test.previewVersion, test.stableVersion)
			} else if !errors.Is(err, test.wantErr) {
				t.Errorf("mismatch, got %v, wanted inclusion of %v", err, test.wantErr)
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

func TestValidateNext(t *testing.T) {
	for _, test := range []struct {
		name           string
		currentVersion string
		nextVersion    string
		wantErr        bool
	}{
		{
			name:        "invalid nextVersion",
			nextVersion: "invalid",
			wantErr:     true,
		},
		{
			name:        "valid nextVersion, no currentVersion",
			nextVersion: "1.2.3",
		},
		{
			name:           "valid nextVersion, invalid currentVersion",
			currentVersion: "invalid",
			nextVersion:    "1.2.3",
			wantErr:        true,
		},
		{
			name:           "nextVersion is earlier than currentVersion",
			currentVersion: "1.3.0",
			nextVersion:    "1.2.0",
			wantErr:        true,
		},
		{
			name:           "nextVersion is equal to currentVersion",
			currentVersion: "1.2.3",
			nextVersion:    "1.2.3",
			wantErr:        true,
		},
		{
			name:           "nextVersion is later than currentVersion",
			currentVersion: "1.2.3",
			nextVersion:    "1.2.4",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateNext(test.currentVersion, test.nextVersion)
			if (err != nil) != test.wantErr {
				t.Errorf("CheckValidNext(%q, %q) error = %v, wantErr %v", test.currentVersion, test.nextVersion, err, test.wantErr)
			}
		})
	}
}
