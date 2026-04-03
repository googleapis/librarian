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
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateMessage(t *testing.T) {
	msg := &api.Message{
		Name:          "Secret",
		Documentation: "A secret message.\nWith two lines.",
		ID:            ".test.Secret",
		Package:       "test",
		Fields: []*api.Field{
			{
				Name:          "secret_key",
				Documentation: "The key.",
			},
		},
	}
	model := api.NewTestAPI([]*api.Message{msg}, []*api.Enum{}, []*api.Service{})
	codec := newTestCodec(t, map[string]string{})
	if err := codec.annotateModel(model); err != nil {
		t.Fatal(err)
	}
	want := &messageAnnotations{
		Name:     "Secret",
		DocLines: []string{"A secret message.", "With two lines."},
	}

	if diff := cmp.Diff(want, msg.Codec, cmpopts.IgnoreFields(messageAnnotations{}, "BoilerPlate", "CopyrightYear")); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}
