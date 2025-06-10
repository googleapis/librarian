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

package cli

import (
	"runtime/debug"
	"strings"
	"time"
)

// Version return the version information for the binary, which is constructed
// following https://go.dev/ref/mod#versions.
func Version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	return version(info)
}

func version(info *debug.BuildInfo) string {
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}

	var revision, at string
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			revision = s.Value
		}
		if s.Key == "vcs.time" {
			at = s.Value
		}
	}

	if revision == "" && at == "" {
		return "not available"
	}

	// Construct the pseudo-version string per
	// https://go.dev/ref/mod#pseudo-versions.
	buf := strings.Builder{}
	buf.WriteString("0.0.0")
	if revision != "" {
		buf.WriteString("-")
		// Per https://go.dev/ref/mod#pseudo-versions, only use the first 12
		// letters of the commit hash.
		buf.WriteString(revision[:12])
	}
	if at != "" {
		// commit time is of the form 2023-01-25T19:57:54Z
		p, err := time.Parse(time.RFC3339, at)
		if err == nil {
			buf.WriteString("-")
			buf.WriteString(p.Format("20060102150405"))
		}
	}
	return buf.String()
}
