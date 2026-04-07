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
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

func TestParseOptions(t *testing.T) {
	cfg := &parser.ModelConfig{
		Codec: map[string]string{
			"copyright-year":        "2038",
			"package-name-override": "GoogleCloudBigtable",
			"root-name":             "test-root",
		},
	}
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	got := newCodec(model, cfg, nil)
	want := &codec{
		GenerationYear: "2038",
		PackageName:    "GoogleCloudBigtable",
		RootName:       "test-root",
		Model:          model,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch in codec (-want, +got)\n:%s", diff)
	}
}

// newTestCodec creates a simple codec for the tests.
func newTestCodec(t *testing.T, model *api.API, options map[string]string) *codec {
	t.Helper()
	cfg := &parser.ModelConfig{
		Codec: options,
	}
	return newCodec(model, cfg, nil)
}
