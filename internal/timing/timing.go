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

// Package timing accumulates named span durations so we can measure where a
// command spends its time and compare the effect of successive changes. A
// Collector is carried on the context; call sites record spans without
// changing their signatures.
//
// A nil *Collector is a no-op, so instrumentation can be added unconditionally:
//
//	defer timing.FromContext(ctx).Span("phase")()
package timing

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type contextKey struct{}

type stat struct {
	count int
	total time.Duration
}

// Collector accumulates named span durations. It is safe for concurrent use so
// that instrumentation survives once generation is parallelized.
type Collector struct {
	mu    sync.Mutex
	stats map[string]*stat
	order []string
	start time.Time
}

// New returns a Collector whose wall-clock starts now.
func New() *Collector {
	return &Collector{stats: make(map[string]*stat), start: time.Now()}
}

// Record adds a single observation of d under name. It is nil-safe.
func (c *Collector) Record(name string, d time.Duration) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	s, ok := c.stats[name]
	if !ok {
		s = &stat{}
		c.stats[name] = s
		c.order = append(c.order, name)
	}
	s.count++
	s.total += d
}

// Span starts timing and returns a function that records the elapsed time under
// name when called. It is nil-safe and intended for deferred use:
//
//	defer c.Span("phase")()
func (c *Collector) Span(name string) func() {
	if c == nil {
		return func() {}
	}
	start := time.Now()
	return func() { c.Record(name, time.Since(start)) }
}

// Summary returns a human-readable table of accumulated spans, sorted by total
// time descending. It is nil-safe (returns "").
func (c *Collector) Summary() string {
	if c == nil {
		return ""
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	wall := time.Since(c.start)
	names := append([]string(nil), c.order...)
	sort.SliceStable(names, func(i, j int) bool {
		return c.stats[names[i]].total > c.stats[names[j]].total
	})
	var b strings.Builder
	fmt.Fprintf(&b, "timing summary (wall %s):\n", wall.Round(time.Millisecond))
	fmt.Fprintf(&b, "  %-36s %8s %14s %14s\n", "phase", "count", "total", "avg")
	for _, n := range names {
		s := c.stats[n]
		var avg time.Duration
		if s.count > 0 {
			avg = s.total / time.Duration(s.count)
		}
		fmt.Fprintf(&b, "  %-36s %8d %14s %14s\n",
			n, s.count, s.total.Round(time.Millisecond), avg.Round(time.Microsecond))
	}
	return b.String()
}

// WithCollector returns a context carrying c so that downstream call sites can
// record spans via FromContext.
func WithCollector(ctx context.Context, c *Collector) context.Context {
	return context.WithValue(ctx, contextKey{}, c)
}

// FromContext returns the Collector on ctx, or nil if none is set. Because a nil
// *Collector is a no-op, the result can be used directly.
func FromContext(ctx context.Context) *Collector {
	c, _ := ctx.Value(contextKey{}).(*Collector)
	return c
}
