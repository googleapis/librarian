// Copyright 2026 Google LLC
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

package serviceconfig

import (
	"path/filepath"
	"testing"
)

// These benchmarks establish a baseline for the per-call cost of reading and
// resolving service configs. serviceconfig.Read/Find are currently invoked
// repeatedly (per API, and several times per API for Java) with no
// memoization; a later PR adds a per-run cache and should move these numbers.

func BenchmarkRead(b *testing.B) {
	path := filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_v1.yaml")
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Read(path); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFind(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Find(googleapisDir, "google/cloud/secretmanager/v1", "java"); err != nil {
			b.Fatal(err)
		}
	}
}
