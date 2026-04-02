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
	libconfig "github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sources"
)

func TestModelAnnotations(t *testing.T) {
	cfg := &parser.ModelConfig{
		SpecificationFormat: libconfig.SpecProtobuf,
		SpecificationSource: "../../testdata/googleapis/google/type",
		Source: &sources.SourceConfig{
			IncludeList: []string{"f1.proto", "f2.proto"},
		},
		Codec: map[string]string{
			"copyright-year": "2026",
		},
	}
	model := api.NewTestAPI(
		[]*api.Message{}, []*api.Enum{},
		[]*api.Service{{Name: "Workflows", Package: "google.cloud.workflows.v1"}})
	codec := newCodec(cfg)
	if err := codec.annotateModel(model, cfg); err != nil {
		t.Fatal(err)
	}
	want := &modelAnnotations{
		PackageName:   "GoogleCloudWorkflowsV1",
		CopyrightYear: "2026",
		Files: []string{
			"../../testdata/googleapis/google/type/f1.proto",
			"../../testdata/googleapis/google/type/f2.proto",
		},
	}
	if diff := cmp.Diff(want, model.Codec, cmpopts.IgnoreFields(modelAnnotations{}, "BoilerPlate")); diff != "" {
		t.Errorf("mismatch in model annotations (-want, +got)\n:%s", diff)
	}
}
