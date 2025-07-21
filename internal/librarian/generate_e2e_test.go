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

//go:build e2e
// +build e2e

package librarian

import "testing"

func TestRunGenerate(t *testing.T) {
	for _, test := range []struct {
		name string
	}{
		{
			name: "test",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Errorf("should not run this test")
		})
	}
}
