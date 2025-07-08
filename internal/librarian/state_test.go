// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package librarian

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLibrarianState_ImageRefAndTag(t *testing.T) {
	for _, test := range []struct {
		name     string
		state    *LibrarianState
		wantName string
		wantTag  string
	}{
		{
			name:  "nil state",
			state: nil,
		},
		{
			name:     "simple tag",
			state:    &LibrarianState{Image: "gcr.io/my-project/my-image:v1.2.3"},
			wantName: "gcr.io/my-project/my-image",
			wantTag:  "v1.2.3",
		},
		{
			name:     "no tag",
			state:    &LibrarianState{Image: "gcr.io/my-project/my-image"},
			wantName: "gcr.io/my-project/my-image",
			wantTag:  "",
		},
		{
			name:     "explicit latest tag",
			state:    &LibrarianState{Image: "ubuntu:latest"},
			wantName: "docker.io/library/ubuntu",
			wantTag:  "latest",
		},
		{
			name:     "with port number",
			state:    &LibrarianState{Image: "my-registry:5000/my/image:v1"},
			wantName: "my-registry:5000/my/image",
			wantTag:  "v1",
		},
		{
			name:     "with port number, no tag",
			state:    &LibrarianState{Image: "my-registry:5000/my/image"},
			wantName: "my-registry:5000/my/image",
			wantTag:  "",
		},
		{
			name:  "empty image string",
			state: &LibrarianState{Image: ""},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotRef, gotTag := test.state.ImageRefAndTag()
			var gotName string
			if gotRef != nil {
				gotName = gotRef.Name()
			}
			if diff := cmp.Diff(test.wantName, gotName); diff != "" {
				t.Errorf("LibrarianState.ImageRefAndTag() name mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantTag, gotTag); diff != "" {
				t.Errorf("LibrarianState.ImageRefAndTag() tag mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
