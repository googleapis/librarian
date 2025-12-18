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
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestGetGcloudType(t *testing.T) {
	for _, test := range []struct {
		name  string
		typez api.Typez
		want  string
	}{
		{"String", api.STRING_TYPE, ""},
		{"Int32", api.INT32_TYPE, "long"},
		{"Int64", api.INT64_TYPE, "long"},
		{"UInt32", api.UINT32_TYPE, "long"},
		{"UInt64", api.UINT64_TYPE, "long"},
		{"Bool", api.BOOL_TYPE, "boolean"},
		{"Float", api.FLOAT_TYPE, "float"},
		{"Double", api.DOUBLE_TYPE, "float"},
		{"Undefined", api.UNDEFINED_TYPE, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := GetGcloudType(test.typez)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("GetGcloudType(%v) mismatch (-want +got):\n%s", test.typez, diff)
			}
		})
	}
}


