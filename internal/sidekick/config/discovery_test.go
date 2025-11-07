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

package config

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	toml "github.com/pelletier/go-toml/v2"
)

func TestLroServices(t *testing.T) {
	for _, test := range []struct {
		Input *Discovery
		Want  map[string]bool
	}{
		{
			Input: &Discovery{
				Pollers: []*Poller{
					{"projects/{project}/zones/{zone}", ".package.zoneOperations.get"},
					{"projects/{project}/regions/{region}", ".package.regionOperations.get"},
				},
			},
			Want: map[string]bool{
				".package.regionOperations": true,
				".package.zoneOperations":   true,
			},
		},
		{
			Input: &Discovery{
				Pollers: []*Poller{
					{"projects/{project}/zones/{zone}", ".package.zoneOperations.get"},
					{"projects/{project}/variation1/{zone}", ".package.zoneOperations.get"},
					{"projects/{project}/variation2/{zone}", ".package.zoneOperations.get"},
					{"projects/{project}/regions/{region}", ".package.regionOperations.get"},
				},
			},
			Want: map[string]bool{
				".package.regionOperations": true,
				".package.zoneOperations":   true,
			},
		},
		{
			Input: &Discovery{},
			Want:  map[string]bool{},
		},
	} {
		got := test.Input.LroServices()
		if diff := cmp.Diff(test.Want, got); diff != "" {
			t.Errorf("mismatch (-want, +got):\n%s", diff)
		}
	}

}

func TestTomlAnnotations(t *testing.T) {
	input := `# Verify the TOML annotations do what we want.
[discovery]
operation-id = '.package.Operation'

[[discovery.pollers]]
prefix = 'prefix-0'
method-id = '.package.zone.get'

[[discovery.pollers]]
prefix = 'prefix-1'
method-id = '.package.region.get'
`
	want := &Discovery{
		OperationID: ".package.Operation",
		Pollers: []*Poller{
			{Prefix: "prefix-0", MethodID: ".package.zone.get"},
			{Prefix: "prefix-1", MethodID: ".package.region.get"},
		},
	}

	var got Config
	err := toml.Unmarshal([]byte(input), &got)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, got.Discovery); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}

func TestPathParameters(t *testing.T) {
	for _, test := range []struct {
		Input *Poller
		Want  []string
	}{
		{Input: &Poller{Prefix: "projects/{project}"}, Want: []string{"project"}},
		{Input: &Poller{Prefix: "projects/{project}/zones/{zone}"}, Want: []string{"project", "zone"}},
		{Input: &Poller{Prefix: "projects/{project}/location/global/"}, Want: []string{"project"}},
		{Input: &Poller{Prefix: "unpexpected/path/without/parameters"}, Want: nil},
	} {
		got := test.Input.PathParameters()
		if diff := cmp.Diff(test.Want, got); diff != "" {
			t.Errorf("mismatch (-want, +got):\n%s", diff)
		}
	}
}

func TestServiceID(t *testing.T) {
	for _, test := range []struct {
		Input *Poller
		Want  string
	}{
		{Input: &Poller{MethodID: ".package.zones.get"}, Want: ".package.zones"},
		{Input: &Poller{MethodID: "bad"}, Want: "bad"},
	} {
		got := test.Input.serviceID()
		if test.Want != got {
			t.Errorf("mismatch want=%s, got=%s", test.Want, got)
		}
	}
}
