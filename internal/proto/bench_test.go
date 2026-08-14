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

package proto

import "testing"

// BenchmarkGather baselines the per-API proto tree walk. It runs once per API
// during generation; the number bounds how much a shared/cached walk could save.
func BenchmarkGather(b *testing.B) {
	const (
		root    = "../testdata/googleapis"
		relPath = "google/cloud/secretmanager/v1"
	)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Gather(root, relPath); err != nil {
			b.Fatal(err)
		}
	}
}
