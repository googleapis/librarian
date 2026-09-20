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

func TestWithSourceLocation(t *testing.T) {
	const (
		file = "google/example/v1/test.proto"
		line = 84
	)
	want := &SourceLocation{File: file, Line: line}

	for _, test := range []struct {
		name  string
		apply func() (*SourceLocation, bool)
	}{
		{
			name: "Service",
			apply: func() (*SourceLocation, bool) {
				s := NewTestService("TestService")
				ret := s.WithSourceLocation(file, line)
				return ret.SourceLocation, ret == s
			},
		},
		{
			name: "Method",
			apply: func() (*SourceLocation, bool) {
				m := NewTestMethod("TestMethod")
				ret := m.WithSourceLocation(file, line)
				return ret.SourceLocation, ret == m
			},
		},
		{
			name: "Message",
			apply: func() (*SourceLocation, bool) {
				m := NewTestMessage("TestMessage")
				ret := m.WithSourceLocation(file, line)
				return ret.SourceLocation, ret == m
			},
		},
		{
			name: "Field",
			apply: func() (*SourceLocation, bool) {
				f := NewTestField("test_field")
				ret := f.WithSourceLocation(file, line)
				return ret.SourceLocation, ret == f
			},
		},
		{
			name: "Enum",
			apply: func() (*SourceLocation, bool) {
				e := NewTestEnum("TestEnum")
				ret := e.WithSourceLocation(file, line)
				return ret.SourceLocation, ret == e
			},
		},
		{
			name: "EnumValue",
			apply: func() (*SourceLocation, bool) {
				ev := NewTestEnumValue("TEST_VALUE", 1)
				ret := ev.WithSourceLocation(file, line)
				return ret.SourceLocation, ret == ev
			},
		},
		{
			name: "OneOf",
			apply: func() (*SourceLocation, bool) {
				o := NewTestOneOf("test_oneof")
				ret := o.WithSourceLocation(file, line)
				return ret.SourceLocation, ret == o
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotLoc, samePtr := test.apply()
			if !samePtr {
				t.Errorf("%s.WithSourceLocation(%q, %d) did not return receiver pointer", test.name, file, line)
			}
			if diff := cmp.Diff(want, gotLoc); diff != "" {
				t.Errorf("%s.WithSourceLocation(%q, %d) mismatch (-want +got):\n%s", test.name, file, line, diff)
			}
		})
	}
}
