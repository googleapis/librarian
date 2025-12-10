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

package serviceconfig

import "testing"

func TestAllowlist(t *testing.T) {
	if len(Allowlist) == 0 {
		t.Fatal("Allowlist should not be empty")
	}

	for _, test := range []struct {
		channel string
		want    bool
	}{
		{"google/cloud/secretmanager/v1", true},
		{"google/cloud/speech/v2", true},
		{"google/iam/v1", true},
		{"grafeas/v1", true},
		{"google/cloud/nonexistent/v1", false},
		{"", false},
	} {
		t.Run(test.channel, func(t *testing.T) {
			if got := Allowlist[test.channel]; got != test.want {
				t.Errorf("Allowlist[%q] = %v, want %v", test.channel, got, test.want)
			}
		})
	}
}
