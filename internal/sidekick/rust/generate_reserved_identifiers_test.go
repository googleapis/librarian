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

package rust

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	libconfig "github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

func TestGenerateModel_ReservedRawIdentifiers(t *testing.T) {
	primaryDataSource := api.NewTestMessage("PrimaryProductDataSource").
		WithPackage("google.shopping.merchant.datasources.v1").
		WithFields(
			api.NewTestField("self").WithType(api.TypezBool),
			api.NewTestField("crate").WithType(api.TypezString),
			api.NewTestField("super").WithType(api.TypezInt32),
		).
		WithOneOfs(
			api.NewTestOneOf("default_rule").WithFields(
				api.NewTestField("self").WithType(api.TypezBool),
				api.NewTestField("crate").WithType(api.TypezString),
				api.NewTestField("super").WithType(api.TypezInt32),
			),
		)

	outDir := t.TempDir()
	model := api.NewTestAPI([]*api.Message{primaryDataSource}, nil, nil).
		WithPackageName("google.shopping.merchant.datasources.v1")
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	cfg := &parser.ModelConfig{
		SpecificationFormat: libconfig.SpecProtobuf,
		Codec: map[string]string{
			"package:wkt":       "source=google.protobuf,package=google-cloud-wkt",
			"template-override": "templates/nosvc",
		},
	}
	if err := Generate(t.Context(), model, outDir, cfg); err != nil {
		t.Fatal(err)
	}

	modelRsBytes, err := os.ReadFile(filepath.Join(outDir, "src", "model.rs"))
	if err != nil {
		t.Fatal(err)
	}
	modelRs := string(modelRsBytes)

	// Verify model.rs doesn't contain invalid raw identifiers
	for _, invalid := range []string{"r#self", "r#Self", "r#crate", "r#super"} {
		if strings.Contains(modelRs, invalid) {
			t.Errorf("model.rs contains invalid raw identifier %q", invalid)
		}
	}

	// Verify oneof enum variant uses Self_
	wantEnumVariant := "Self_(bool)"
	if !strings.Contains(modelRs, wantEnumVariant) {
		t.Errorf("model.rs missing oneof variant %q", wantEnumVariant)
	}

	// Verify oneof branch accessor methods use self_, crate_, super_
	for _, wantAccessor := range []string{
		"pub fn self_(&self) -> std::option::Option<&bool>",
		"pub fn crate_(&self) -> std::option::Option<&std::string::String>",
		"pub fn super_(&self) -> std::option::Option<&i32>",
	} {
		if !strings.Contains(modelRs, wantAccessor) {
			t.Errorf("model.rs missing accessor %q", wantAccessor)
		}
	}

	// Verify setters
	for _, wantSetter := range []string{
		"pub fn set_self<",
		"pub fn set_crate<",
		"pub fn set_super<",
	} {
		if !strings.Contains(modelRs, wantSetter) {
			t.Errorf("model.rs missing setter %q", wantSetter)
		}
	}

	// Verify struct field definitions use self_, crate_, super_
	for _, wantField := range []string{
		"pub self_: bool",
		"pub crate_: std::string::String",
		"pub super_: i32",
	} {
		if !strings.Contains(modelRs, wantField) {
			t.Errorf("model.rs missing field %q", wantField)
		}
	}
}
