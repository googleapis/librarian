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

func TestAnnotateOneOf(t *testing.T) {
	oneof := api.NewTestOneOf("test_alternatives").
		WithDocumentation("A test oneof.")
	message := api.NewTestMessage("TestMessage").
		WithPackage("google.cloud.test.v1").
		WithOneOfs(oneof)
	model := api.NewTestAPI([]*api.Message{message}, nil, nil).
		WithPackageName("google.cloud.test.v1")
	codec := newTestCodec(t, model, nil)
	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	want := &oneOfAnnotations{
		Name:         "TestAlternativesOneOf",
		PropertyName: "testAlternatives",
		DocLines:     []string{"A test oneof."},
		Checker:      "testAlternativesCheckAndSet",
	}

	if diff := cmp.Diff(want, oneof.Codec); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
