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

package api

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLroServices(t *testing.T) {
	d := &Discovery{
		Pollers: []*Poller{
			{Prefix: "p1", MethodID: "Service1.Method1"},
			{Prefix: "p2", MethodID: "Service1.Method2"},
			{Prefix: "p3", MethodID: "Service2.Method3"},
		},
	}
	want := map[string]bool{
		"Service1": true,
		"Service2": true,
	}
	got := d.LroServices()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("LroServices() mismatch (-want +got):\n%s", diff)
	}
}

func TestPathParameters(t *testing.T) {
	for _, test := range []struct {
		input *Poller
		want  []string
	}{
		{
			input: &Poller{Prefix: "projects/{project}/zones/{zone}"},
			want:  []string{"project", "zone"},
		},
		{
			input: &Poller{Prefix: "abc/def"},
			want:  nil,
		},
		{
			input: &Poller{Prefix: "a/{b}"},
			want:  []string{"b"},
		},
	} {
		got := test.input.PathParameters()
		if diff := cmp.Diff(test.want, got); diff != "" {
			t.Errorf("PathParameters(%q) mismatch (-want +got):\n%s", test.input.Prefix, diff)
		}
	}
}
