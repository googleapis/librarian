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

package gcloud

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDeriveAPIPath(t *testing.T) {
	for _, test := range []struct {
		name string
		want string
	}{
		{"accessapproval", "google/cloud/accessapproval/v1"},
		{"secretmanager", "google/cloud/secretmanager/v1"},
		{"parallelstore", "google/cloud/parallelstore/v1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := DeriveAPIPath(test.name)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDefaultLibraryName(t *testing.T) {
	for _, test := range []struct {
		name string
		api  string
		want string
	}{
		{"google-cloud-v1", "google/cloud/accessapproval/v1", "accessapproval"},
		{"google-cloud-v1beta", "google/cloud/secretmanager/v1beta2", "secretmanager"},
		{"google-no-cloud", "google/longrunning/v1", "longrunning"},
		{"unknown-shape", "foo/bar/baz", "foo"},
		{"empty", "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := DefaultLibraryName(test.api)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDefaultOutput(t *testing.T) {
	for _, test := range []struct {
		name          string
		libName       string
		defaultOutput string
		want          string
	}{
		{"empty default falls back to generated", "accessapproval", "", "generated/accessapproval"},
		{"explicit default", "accessapproval", "out", "out/accessapproval"},
		{"explicit default same as fallback", "accessapproval", "generated", "generated/accessapproval"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := DefaultOutput(test.libName, test.defaultOutput)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
