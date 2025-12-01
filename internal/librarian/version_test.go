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

package librarian

import (
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
			name: "pseudoversion",
			want: fmt.Sprintf("%s-123456789000-20230125195754", baseVersion),
			buildinfo: &debug.BuildInfo{
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "1234567890001234"},
					{Key: "vcs.time", Value: "2023-01-25T19:57:54Z"},
				},
			},
		},
		{
			name: "pseudoversion only revision",
			want: fmt.Sprintf("%s-123456789000", baseVersion),
			buildinfo: &debug.BuildInfo{
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "1234567890001234"},
				},
			},
		},
		{
			name: "pseudoversion only time",
			want: fmt.Sprintf("%s-20230102150405", baseVersion),
			buildinfo: &debug.BuildInfo{
				Settings: []debug.BuildSetting{
					{Key: "vcs.time", Value: "2023-01-02T15:04:05Z"},
				},
			},
		},
		{
			name: "pseudoversion invalid time",
			want: fmt.Sprintf("%s-123456789000", baseVersion),
			buildinfo: &debug.BuildInfo{
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "123456789000"},
					{Key: "vcs.time", Value: "invalid-time"},
				},
			},
		},
		{
			name: "revision less than 12 chars",
			want: fmt.Sprintf("%s-shortrev-20230125195754", baseVersion),
			buildinfo: &debug.BuildInfo{
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "shortrev"},
					{Key: "vcs.time", Value: "2023-01-25T19:57:54Z"},
				},
			},
		},
		{
			name:      "local development",
			want:      "not available",
			buildinfo: &debug.BuildInfo{},
		},
		{
			name: "retracted version, build from tagged commit",
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
		{
			name: "retracted version, local dev builds from untagged commits",
			want: fmt.Sprintf("%s-123456789000-20230125195754", baseVersion),
			buildinfo: &debug.BuildInfo{
				Main: debug.Module{
					Version: "v1.0.2-0.20251125150633-68dcc1cc4ab4+dirty",
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
