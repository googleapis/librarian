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

package utils

import "strings"

// InferTrackFromPackage infers the release track from the proto package name.
// as mandated per AIP-185
// e.g. "google.cloud.parallelstore.v1beta" -> "beta".
func InferTrackFromPackage(pkg string) string {
	parts := strings.Split(pkg, ".")

	version := parts[len(parts)-1]

	if strings.Contains(version, "alpha") {
		return "alpha"
	}
	if strings.Contains(version, "beta") {
		return "beta"
	}
	return "ga"
}
