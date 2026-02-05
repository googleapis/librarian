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

package librarian

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	baseVersion := strings.TrimSpace(versionString)
	for _, test := range []struct {
		name      string
		want      string
		buildinfo *debug.BuildInfo
	}{
		{
			name: "tagged version",
			want: "1.2.3",
			buildinfo: &debug.BuildInfo{
				Main: debug.Module{
					Version: "1.2.3",
				},
			},
		},
		{
			name:      "local development",
			want:      versionNotAvailable,
			buildinfo: &debug.BuildInfo{},
		},
		{
			name: "local development with VCS info",
			want: versionNotAvailable,
			buildinfo: &debug.BuildInfo{
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "1234567890001234"},
					{Key: "vcs.time", Value: "2023-01-25T19:57:54Z"},
				},
			},
		},
		{
			name: "local development with dirty suffix",
			want: versionNotAvailable,
			buildinfo: &debug.BuildInfo{
				Main: debug.Module{
					Version: "v1.0.2-0.20260130024826-f525c91d74e9+dirty",
				},
			},
		},
		{
			name: "retracted version",
			want: fmt.Sprintf("%s-123456789000-20230125195754", baseVersion),
			buildinfo: &debug.BuildInfo{
				Main: debug.Module{
					Version: "v1.0.0",
				},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "1234567890001234"},
					{Key: "vcs.time", Value: "2023-01-25T19:57:54Z"},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := version(test.buildinfo); got != test.want {
				t.Errorf("got %s; want %s", got, test.want)
			}
		})
	}
}

func TestNewPseudoVersion(t *testing.T) {
	baseVersion := strings.TrimSpace(versionString)
	for _, test := range []struct {
		name      string
		want      string
		buildinfo *debug.BuildInfo
	}{
		{
			name: "full pseudo-version",
			want: fmt.Sprintf("%s-123456789012-20230125195754", baseVersion),
			buildinfo: &debug.BuildInfo{
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "1234567890123456"},
					{Key: "vcs.time", Value: "2023-01-25T19:57:54Z"},
				},
			},
		},
		{
			name: "only revision",
			want: fmt.Sprintf("%s-123456789012", baseVersion),
			buildinfo: &debug.BuildInfo{
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "1234567890123456"},
				},
			},
		},
		{
			name: "only time",
			want: fmt.Sprintf("%s-20230125195754", baseVersion),
			buildinfo: &debug.BuildInfo{
				Settings: []debug.BuildSetting{
					{Key: "vcs.time", Value: "2023-01-25T19:57:54Z"},
				},
			},
		},
		{
			name: "invalid time format",
			want: fmt.Sprintf("%s-123456789012", baseVersion),
			buildinfo: &debug.BuildInfo{
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "1234567890123456"},
					{Key: "vcs.time", Value: "invalid-time"},
				},
			},
		},
		{
			name: "short revision",
			want: fmt.Sprintf("%s-abc123-20230125195754", baseVersion),
			buildinfo: &debug.BuildInfo{
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "abc123"},
					{Key: "vcs.time", Value: "2023-01-25T19:57:54Z"},
				},
			},
		},
		{
			name:      "no VCS info",
			want:      versionNotAvailable,
			buildinfo: &debug.BuildInfo{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := newPseudoVersion(test.buildinfo); got != test.want {
				t.Errorf("got %s; want %s", got, test.want)
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	for _, test := range []struct {
		name          string
		configVersion string
		binaryVersion string
		wantErr       error
	}{
		{
			name:          "matching versions",
			configVersion: "v1.0.0",
			binaryVersion: "v1.0.0",
		},
		{
			name:          "mismatched versions",
			configVersion: "v1.0.0",
			binaryVersion: "v2.0.0",
			wantErr:       errVersionMismatch,
		},
		{
			name:          "local build skips check",
			configVersion: "v1.0.0",
			binaryVersion: versionNotAvailable,
		},
		{
			name:          "empty config version",
			configVersion: "",
			binaryVersion: "v1.0.0",
			wantErr:       errNoConfigVersion,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := compareVersions(test.configVersion, test.binaryVersion)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("got %v; want %v", err, test.wantErr)
			}
		})
	}
}

func TestSkipVersionCheck(t *testing.T) {
	for _, test := range []struct {
		name   string
		setKey bool
		value  bool
		want   bool
	}{
		{
			name:   "key set to true",
			setKey: true,
			value:  true,
			want:   true,
		},
		{
			name:   "key set to false",
			setKey: true,
			value:  false,
			want:   false,
		},
		{
			name:   "key not set",
			setKey: false,
			want:   false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()
			if test.setKey {
				ctx = context.WithValue(ctx, skipVersionCheckKey{}, test.value)
			}
			if got := skipVersionCheck(ctx); got != test.want {
				t.Errorf("got %v; want %v", got, test.want)
			}
		})
	}
}
