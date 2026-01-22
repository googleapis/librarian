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

package sidekick

import (
	"context"
	"os"
	"path"
	"testing"

	"cloud.google.com/go/sdk/internal/testhelper"
)

func TestRustProstFromProtobuf(t *testing.T) {
	if testing.Short() {
		t.Skip("slow test: Rust Prost code generation")
	}
	testhelper.RequireCommand(t, "cargo")
	testhelper.RequireCommand(t, "protoc")
	outDir := t.TempDir()
	svcConfig := path.Join(googleapisRoot, "google/type/type.yaml")
	specificationSource := path.Join(googleapisRoot, "google/type")

	cmdLine := &CommandLine{
		Command:             []string{},
		SpecificationFormat: "protobuf",
		SpecificationSource: specificationSource,
		Source: map[string]string{
			"googleapis-root": googleapisRoot,
		},
		ServiceConfig: svcConfig,
		Language:      "rust+prost",
		Output:        outDir,
		Codec: map[string]string{
			"copyright-year":      "2025",
			"not-for-publication": "true",
		},
	}
	cmdGenerate, _, _ := cmdSidekick.lookup([]string{"generate"})
	if err := runCommand(context.Background(), cmdGenerate, cmdLine); err != nil {
		t.Fatal(err)
	}

	for _, expected := range []string{".sidekick.toml", "google.r#type.rs"} {
		filename := path.Join(outDir, expected)
		stat, err := os.Stat(filename)
		if os.IsNotExist(err) {
			t.Errorf("missing %s: %s", filename, err)
		}
		if stat.Mode().Perm()|0666 != 0666 {
			t.Errorf("generated files should not be executable %s: %o", filename, stat.Mode())
		}
	}
}
