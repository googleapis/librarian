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

func TestGetVerb(t *testing.T) {
	for _, test := range []struct {
		name       string
		methodName string
		want       string
		wantErr    bool
	}{
		{"Get", "GetInstance", "describe", false},
		{"List", "ListInstances", "list", false},
		{"Create", "CreateInstance", "create", false},
		{"Update", "UpdateInstance", "update", false},
		{"Delete", "DeleteInstance", "delete", false},
		{"Custom", "DetachDisk", "detach_disk", false},
		{"Empty", "", "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := GetVerb(test.methodName)
			if (err != nil) != test.wantErr {
				t.Errorf("GetVerb(%q) error = %v, wantErr %v", test.methodName, err, test.wantErr)
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("GetVerb(%q) mismatch (-want +got):\n%s", test.methodName, diff)
			}
		})
	}
}
