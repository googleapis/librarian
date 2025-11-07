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

package rustrelease

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFindManifest(t *testing.T) {
	setupForVersionBump(t, "find-manifest-0.0.1")
	// A hypothetical set of file changes.
	input := []string{
		"doc/not-in-a-Cargo-file",
		"doc/subdir/not-a-Cargo-file-either.md",
		"src/storage/src/lib.rs",
		"src/storage/src/generated/model.rs",
		"src/storage/src/generated/stub.rs",
		"src/storage/src/generated/client.rs",
		"src/generated/cloud/secretmanager/v1/src/stub/subdir/file.rs",
		"src/gax-internal/echo-server/src/lib.rs",
	}
	got := findCargoManifests(input)
	want := []string{
		"src/gax-internal/echo-server/Cargo.toml",
		"src/generated/cloud/secretmanager/v1/Cargo.toml",
		"src/storage/Cargo.toml",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("generated files changed mismatch (-want +got):\n%s", diff)
	}
}
