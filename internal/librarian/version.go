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
	"runtime/debug"
	"strings"
)

// versionDevel is the version string for local builds without a module version.
const versionDevel = "devel"

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
	// Local development builds have no proper version tag, or have
	// uncommitted changes indicated by the +dirty suffix.
	if info.Main.Version == "" || info.Main.Version == "(devel)" ||
		strings.HasSuffix(info.Main.Version, "+dirty") {
		return versionDevel
	}
	return info.Main.Version
}
