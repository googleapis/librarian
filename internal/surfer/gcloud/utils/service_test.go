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

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestInferTrackFromPackage(t *testing.T) {
	for _, test := range []struct {
		name string
		pkg  string
		want string
	}{
		{"GA package", "google.cloud.parallelstore.v1", "ga"},
		{"Beta package", "google.cloud.parallelstore.v1beta", "beta"},
		{"Alpha package", "google.cloud.parallelstore.v1alpha", "alpha"},
		{"Empty package", "", "ga"},
		{"Package without version", "google.cloud.parallelstore", "ga"},
		{"Other version", "google.cloud.parallelstore.v2", "ga"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := InferTrackFromPackage(test.pkg)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("InferTrackFromPackage mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
