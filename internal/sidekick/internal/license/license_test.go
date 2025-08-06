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

package license

import (
	"strings"
	"testing"
)

func TestLicense(t *testing.T) {
	got := LicenseHeader("2038")
	want := "Copyright 2038"
	if len(got) != 13 {
		t.Errorf("bad header length from LicenseHeader(), got=%d, want=%q", len(got), 13)
	}
	if !strings.Contains(got[0], want) {
		t.Errorf("bad start line for LicenseHeader(), got=%q, want=%q", got[0], want)
	}
}
