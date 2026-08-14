// Copyright 2026 Google LLC
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
	"errors"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/timing"
)

func TestForEachLibraryRunsAllAndRecordsSpans(t *testing.T) {
	libs := []*config.Library{{Name: "a"}, {Name: "b"}, {Name: "c"}}
	tc := timing.New("test")
	ctx := timing.WithCollector(context.Background(), tc)

	var seen sync.Map
	var count atomic.Int64
	err := forEachLibrary(ctx, libs, 2, timing.PhaseGenerateLibrary, func(_ context.Context, lib *config.Library) error {
		seen.Store(lib.Name, true)
		count.Add(1)
		return nil
	})
	if err != nil {
		t.Fatalf("forEachLibrary returned error: %v", err)
	}
	if count.Load() != int64(len(libs)) {
		t.Errorf("ran %d libraries, want %d", count.Load(), len(libs))
	}
	for _, lib := range libs {
		if _, ok := seen.Load(lib.Name); !ok {
			t.Errorf("library %q was not processed", lib.Name)
		}
	}
	if got := tc.Summary(); !strings.Contains(got, timing.PhaseGenerateLibrary) {
		t.Errorf("expected a %q span in the summary; got:\n%s", timing.PhaseGenerateLibrary, got)
	}
}

func TestForEachLibraryPropagatesError(t *testing.T) {
	libs := []*config.Library{{Name: "a"}, {Name: "b"}}
	wantErr := errors.New("boom")
	err := forEachLibrary(context.Background(), libs, 4, timing.PhaseGenerateLibrary, func(_ context.Context, lib *config.Library) error {
		if lib.Name == "b" {
			return wantErr
		}
		return nil
	})
	if !errors.Is(err, wantErr) {
		t.Errorf("forEachLibrary error = %v, want %v", err, wantErr)
	}
}

func TestConcurrencyLimit(t *testing.T) {
	cpus := runtime.NumCPU()
	for _, test := range []struct {
		name string
		jobs int
		want int
	}{
		{name: "zero falls back to NumCPU", jobs: 0, want: cpus},
		{name: "negative falls back to NumCPU", jobs: -4, want: cpus},
		{name: "positive is used as-is", jobs: 3, want: 3},
		{name: "one", jobs: 1, want: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := concurrencyLimit(test.jobs); got != test.want {
				t.Errorf("concurrencyLimit(%d) = %d, want %d", test.jobs, got, test.want)
			}
		})
	}
}
