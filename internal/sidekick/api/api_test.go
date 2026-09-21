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

package api

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSourceLocation(t *testing.T) {
	got := SourceLocation{
		Filename: "google/cloud/test/v1/test.proto",
		Line:     42,
	}
	want := SourceLocation{
		Filename: "google/cloud/test/v1/test.proto",
		Line:     42,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAPIDefinitionLocation(t *testing.T) {
	t.Run("nil api", func(t *testing.T) {
		var a *API
		loc, ok := a.DefinitionLocation("any")
		if ok || loc != (SourceLocation{}) {
			t.Errorf("expected zero SourceLocation and false on nil API, got %v, %v", loc, ok)
		}
	})

	t.Run("empty api without map", func(t *testing.T) {
		a := &API{}
		loc, ok := a.DefinitionLocation("any")
		if ok || loc != (SourceLocation{}) {
			t.Errorf("expected zero SourceLocation and false on empty API, got %v, %v", loc, ok)
		}
	})

	t.Run("add and retrieve definition locations", func(t *testing.T) {
		a := &API{}
		wantLoc1 := SourceLocation{Filename: "test.proto", Line: 10}
		wantLoc2 := SourceLocation{Filename: "test.proto", Line: 25}

		a.AddDefinitionLocation(".test.MyService", wantLoc1)
		a.AddDefinitionLocation(".test.MyMessage", wantLoc2)

		gotLoc1, ok1 := a.DefinitionLocation(".test.MyService")
		if !ok1 {
			t.Fatal("expected to find .test.MyService")
		}
		if diff := cmp.Diff(wantLoc1, gotLoc1); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}

		gotLoc2, ok2 := a.DefinitionLocation(".test.MyMessage")
		if !ok2 {
			t.Fatal("expected to find .test.MyMessage")
		}
		if diff := cmp.Diff(wantLoc2, gotLoc2); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}

		_, ok3 := a.DefinitionLocation(".test.NonExistent")
		if ok3 {
			t.Errorf("expected false for non-existent symbol")
		}
	})

	t.Run("fluent builder WithDefinitionLocation", func(t *testing.T) {
		a := NewTestAPI(nil, nil, nil).
			WithDefinitionLocation(".test.Foo", "foo.proto", 15).
			WithDefinitionLocation(".test.Bar", "bar.proto", 30)

		wantFoo := SourceLocation{Filename: "foo.proto", Line: 15}
		gotFoo, ok := a.DefinitionLocation(".test.Foo")
		if !ok {
			t.Fatal("expected to find .test.Foo")
		}
		if diff := cmp.Diff(wantFoo, gotFoo); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}

		wantBar := SourceLocation{Filename: "bar.proto", Line: 30}
		gotBar, ok := a.DefinitionLocation(".test.Bar")
		if !ok {
			t.Fatal("expected to find .test.Bar")
		}
		if diff := cmp.Diff(wantBar, gotBar); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("leading dot normalization", func(t *testing.T) {
		a := &API{}
		locWithDot := SourceLocation{Filename: "with_dot.proto", Line: 10}
		locWithoutDot := SourceLocation{Filename: "without_dot.proto", Line: 20}

		// Register one with a leading dot and one without.
		a.AddDefinitionLocation(".test.WithDot", locWithDot)
		a.AddDefinitionLocation("test.WithoutDot", locWithoutDot)

		for _, test := range []struct {
			query string
			want  SourceLocation
		}{
			{query: ".test.WithDot", want: locWithDot},
			{query: "test.WithDot", want: locWithDot},
			{query: "test.WithoutDot", want: locWithoutDot},
			{query: ".test.WithoutDot", want: locWithoutDot},
		} {
			t.Run(test.query, func(t *testing.T) {
				got, ok := a.DefinitionLocation(test.query)
				if !ok {
					t.Fatalf("expected to find %q", test.query)
				}
				if diff := cmp.Diff(test.want, got); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			})
		}

		for _, nonExistent := range []string{".test.Missing", "test.Missing"} {
			if got, ok := a.DefinitionLocation(nonExistent); ok || got != (SourceLocation{}) {
				t.Errorf("expected zero SourceLocation and false for %q, got %v, %v", nonExistent, got, ok)
			}
		}
	})
}
