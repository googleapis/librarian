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

package librarian

import (
	"context"
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

func TestFetchSource(t *testing.T) {
	ctx := context.Background()

	t.Run("nil source", func(t *testing.T) {
		dir, err := fetchSource(ctx, nil, "some-repo")
		if err != nil {
			t.Errorf("fetchSource() error = %v, wantErr %v", err, false)
		}
		if dir != "" {
			t.Errorf("fetchSource() = %q, want %q", dir, "")
		}
	})

	t.Run("source with dir", func(t *testing.T) {
		wantDir := "local/dir"
		source := &config.Source{Dir: wantDir}
		gotDir, err := fetchSource(ctx, source, "some-repo")
		if err != nil {
			t.Errorf("fetchSource() error = %v, wantErr %v", err, false)
		}
		if gotDir != wantDir {
			t.Errorf("fetchSource() = %q, want %q", gotDir, wantDir)
		}
	})
}
