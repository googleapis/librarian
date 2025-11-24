// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gapicyaml

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestConfig(t *testing.T) {
	for _, test := range []struct {
		name string
		file string
	}{
		{"logging", "testdata/logging_gapic.yaml"},
		{"bigtable", "testdata/bigtableadmin_gapic.yaml"},
	} {
		t.Run(test.name, func(t *testing.T) {
			want, err := Read(test.file)
			if err != nil {
				t.Fatal(err)
			}
			tmpfile := t.TempDir() + "/test_gapic.yaml"
			if err := want.Write(tmpfile); err != nil {
				t.Fatal(err)
			}
			got, err := Read(tmpfile)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
