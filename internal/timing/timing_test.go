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

package timing

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRecordAggregates(t *testing.T) {
	c := New()
	c.Record("a", 10*time.Millisecond)
	c.Record("a", 30*time.Millisecond)
	c.Record("b", 5*time.Millisecond)

	c.mu.Lock()
	defer c.mu.Unlock()
	if got := c.stats["a"].count; got != 2 {
		t.Errorf("stats[a].count = %d, want 2", got)
	}
	if got := c.stats["a"].total; got != 40*time.Millisecond {
		t.Errorf("stats[a].total = %s, want 40ms", got)
	}
	if got := c.stats["b"].count; got != 1 {
		t.Errorf("stats[b].count = %d, want 1", got)
	}
}

func TestSpanRecordsElapsed(t *testing.T) {
	c := New()
	stop := c.Span("work")
	stop()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stats["work"].count != 1 {
		t.Fatalf("span did not record a single observation: %+v", c.stats["work"])
	}
}

func TestNilCollectorIsNoOp(t *testing.T) {
	var c *Collector
	// None of these must panic.
	c.Record("a", time.Second)
	c.Span("a")()
	if got := c.Summary(); got != "" {
		t.Errorf("nil Summary() = %q, want empty", got)
	}
}

func TestFromContextRoundTrip(t *testing.T) {
	c := New()
	ctx := WithCollector(context.Background(), c)
	if FromContext(ctx) != c {
		t.Error("FromContext did not return the stored collector")
	}
	if FromContext(context.Background()) != nil {
		t.Error("FromContext on a bare context should be nil")
	}
}

func TestSummaryContainsPhases(t *testing.T) {
	c := New()
	c.Record("alpha", 2*time.Millisecond)
	c.Record("beta", 1*time.Millisecond)
	got := c.Summary()
	for _, want := range []string{"timing summary", "alpha", "beta", "count"} {
		if !strings.Contains(got, want) {
			t.Errorf("Summary() missing %q; got:\n%s", want, got)
		}
	}
}

func TestConcurrentRecord(t *testing.T) {
	c := New()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Record("shared", time.Millisecond)
		}()
	}
	wg.Wait()
	c.mu.Lock()
	defer c.mu.Unlock()
	if got := c.stats["shared"].count; got != 50 {
		t.Errorf("concurrent count = %d, want 50", got)
	}
}
