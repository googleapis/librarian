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

package swift

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestLookupMessage(t *testing.T) {
	msg := &api.Message{Name: "Secret", ID: ".test.Secret"}
	model := api.NewTestAPI([]*api.Message{msg}, []*api.Enum{}, []*api.Service{})

	got, err := lookupMessage(model, ".test.Secret")
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(msg, got); diff != "" {
		t.Errorf("lookupMessage() mismatch (-want +got):\n%s", diff)
	}
}

func TestLookupMessage_Error(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})

	_, err := lookupMessage(model, ".test.Missing")
	if err == nil {
		t.Errorf("lookupMessage() expected error, got nil")
	}
}
