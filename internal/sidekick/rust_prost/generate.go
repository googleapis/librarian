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

package rust_prost

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/config"
	"github.com/googleapis/librarian/internal/sidekick/language"
)

//go:embed all:templates
var templates embed.FS

// Generate generates Rust code from the model using prost.
func Generate(model *api.API, outdir string, cfg *config.Config) error {
	if cfg.General.SpecificationFormat != "protobuf" {
		return fmt.Errorf("the `rust+prost` generator only supports `protobuf` as a specification source, outdir=%s", outdir)
	}
	if err := command.Run("cargo", "--version"); err != nil {
		return fmt.Errorf("got an error trying to run `cargo --version`, the instructions on https://www.rust-lang.org/learn/get-started may solve this problem: %w", err)
	}
	if err := command.Run("protoc", "--version"); err != nil {
		return fmt.Errorf("got an error trying to run `protoc --version`, the instructions on https://grpc.io/docs/protoc-installation/ may solve this problem: %w", err)
	}

	codec := newCodec(cfg)
	codec.annotateModel(model, cfg)
	provider := templatesProvider()
	generatedFiles := language.WalkTemplatesDir(templates, "templates/prost")
	tmpDir, err := os.MkdirTemp("", "rust-prost-*")
	if err != nil {
		return fmt.Errorf("cannot create temporary directory for rust+prost output: %w", err)
	}
	defer os.RemoveAll(tmpDir)
	if err := language.GenerateFromModel(tmpDir, model, provider, generatedFiles); err != nil {
		return err
	}
	rootName := cfg.Source[codec.RootName]
	return buildRS(rootName, tmpDir, outdir)
}

func templatesProvider() language.TemplateProvider {
	return func(name string) (string, error) {
		contents, err := templates.ReadFile(name)
		if err != nil {
			return "", err
		}
		return string(contents), nil
	}
}

func buildRS(rootName, tmpDir, outDir string) error {
	absRoot, err := filepath.Abs(rootName)
	if err != nil {
		return err
	}
	absOutDir, err := filepath.Abs(outDir)
	if err != nil {
		return err
	}
	cmd := exec.Command("cargo", "build", "--features", "_generate-protos")
	cmd.Dir = tmpDir
	cmd.Env = append(os.Environ(), fmt.Sprintf("SOURCE_ROOT=%s", absRoot))
	cmd.Env = append(cmd.Env, fmt.Sprintf("DEST=%s", absOutDir))
	fmt.Fprintf(os.Stderr, "Running: %s\n", cmd.String())
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%v: %v\n%s", cmd, err, output)
	}
	return nil
}
