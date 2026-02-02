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

package common

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestFetchSource(t *testing.T) {
	ctx := t.Context()
	for _, tt := range []struct {
		name    string
		source  *config.Source
		wantDir string
		wantErr bool
	}{
		{
			name:    "nil source",
			source:  nil,
			wantDir: "",
		},
		{
			name:    "source with dir",
			source:  &config.Source{Dir: "local/dir"},
			wantDir: "local/dir",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			gotDir, err := fetchSource(ctx, tt.source, "some-repo")
			if (err != nil) != tt.wantErr {
				t.Errorf("fetchSource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.wantDir, gotDir); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
