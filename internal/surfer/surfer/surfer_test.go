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

package surfer

import (
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	for _, test := range []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name: "valid command",
			args: []string{
				"surfer",
				"generate",
				"../gcloud/testdata/parallelstore/gcloud.yaml",
				"--out", "../gcloud/testdata/parallelstore/surface",
			},
		},
		{
			name: "invalid gcloud.yaml filepath",
			args: []string{
				"surfer",
				"generate",
				"invalidpath/gcloud.yaml",
			},
			wantErr: true,
		},
		{
			name:    "missing config arg",
			args:    []string{"surfer", "generate"},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := Run(t.Context(), test.args...); err != nil {
				// TODO(https://github.com/googleapis/librarian/issues/2817):
				// remove once the generate functionality has been implemented
				if strings.Contains(err.Error(), "failed to create API model") {
					return
				}
				if !test.wantErr {
					t.Fatal(err)
				}
			}
		})
	}
}