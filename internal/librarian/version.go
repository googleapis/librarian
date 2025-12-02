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
	_ "embed"
	"runtime/debug"
	"strings"
	"time"
)

//go:embed version.txt
var versionString string

// retractedVersions is a list of Go module versions that have been officially retracted
// via the go.mod 'retract' directive. v1.0.2 added to account for local dev builds
// from untagged commits.
var retractedVersions = []string{"v1.0.0", "v1.0.1", "v1.0.2"}

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
	isRetracted := false
	for _, v := range retractedVersions {
		if strings.HasPrefix(info.Main.Version, v) {
			isRetracted = true
			break
		}
	}
	// A pseudo-version should be used for retracted versions or for
	// development builds that don't have a proper version tag.
	isDevelBuild := info.Main.Version == "" || info.Main.Version == "(devel)"
	if isRetracted || isDevelBuild {
		return newPseudoVersion(info)
	}
	return info.Main.Version
}

// newPseudoVersion constructs a pseudo-version string from the build info.
func newPseudoVersion(info *debug.BuildInfo) string {
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
	var buf strings.Builder
	buf.WriteString(strings.TrimSpace(versionString))
	if revision != "" {
		buf.WriteString("-")
		// Per https://go.dev/ref/mod#pseudo-versions, only use the first 12
		// letters of the commit hash.
		if len(revision) > 12 {
			revision = revision[:12]
		}
		buf.WriteString(revision)
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
