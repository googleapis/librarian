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

	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestGetGcloudType(t *testing.T) {
	for _, test := range []struct {
		name  string
		typez api.Typez
		want  string
	}{
		{"String", api.STRING_TYPE, "str"},
		{"Int32", api.INT32_TYPE, "int"},
		{"Int64", api.INT64_TYPE, "long"},
		{"UInt32", api.UINT32_TYPE, "int"},
		{"UInt64", api.UINT64_TYPE, "long"},
		{"Bool", api.BOOL_TYPE, "bool"},
		{"Float", api.FLOAT_TYPE, "float"},
		{"Double", api.DOUBLE_TYPE, "float"},
		{"Bytes", api.BYTES_TYPE, "bytes"},
		{"Enum", api.ENUM_TYPE, "str"},
		{"Message", api.MESSAGE_TYPE, "arg_object"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := GetGcloudType(test.typez)
			if got != test.want {
				t.Errorf("GetGcloudType(%v) = %q, want %q", test.typez, got, test.want)
			}
		})
	}
}
