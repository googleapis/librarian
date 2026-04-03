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

func TestAnnotateField(t *testing.T) {
	field := &api.Field{
		Name:          "secret_payload",
		Documentation: "The secret version payload.",
		ID:            ".test.SecretVersion.secret_payload",
	}
	msg := &api.Message{
		Name:    "Secret",
		ID:      ".test.SecretVersion",
		Package: "test",
		Fields:  []*api.Field{field},
	}
	model := api.NewTestAPI([]*api.Message{msg}, []*api.Enum{}, []*api.Service{})
	codec := newTestCodec(t, model, map[string]string{})
	if err := codec.annotateModel(model); err != nil {
		t.Fatal(err)
	}
	want := &fieldAnnotations{
		Name:     "secretPayload",
		DocLines: []string{"The secret version payload."},
	}

	if diff := cmp.Diff(want, field.Codec); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}
