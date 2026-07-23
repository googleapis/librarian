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
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	libconfig "github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"

	"github.com/googleapis/librarian/internal/sidekick/language"
)

//go:embed all:templates
var templates embed.FS

// Generate generates Rust code from the model using prost.
func Generate(ctx context.Context, model *api.API, outdir string, template string, cfg *parser.ModelConfig) error {
	if cfg.SpecificationFormat != libconfig.SpecProtobuf {
		return fmt.Errorf("the `rust+prost` generator only supports `protobuf` as a specification source, outdir=%s", outdir)
	}
	if err := command.Run(ctx, command.Cargo, "--version"); err != nil {
		return fmt.Errorf("got an error trying to run `cargo --version`, the instructions on https://www.rust-lang.org/learn/get-started may solve this problem: %w", err)
	}
	if err := command.Run(ctx, "protoc", "--version"); err != nil {
		return fmt.Errorf("got an error trying to run `protoc --version`, the instructions on https://grpc.io/docs/protoc-installation/ may solve this problem: %w", err)
	}

	codec := newCodec(cfg)
	if err := codec.annotateModel(model, cfg); err != nil {
		return fmt.Errorf("annotating model: %w", err)
	}
	provider := templatesProvider()
	generatedFiles := language.WalkTemplatesDir(templates, "templates/"+template)
	tmpDir, err := os.MkdirTemp("", "rust-prost-*")
	if err != nil {
		return fmt.Errorf("cannot create temporary directory for rust+prost output: %w", err)
	}
	defer os.RemoveAll(tmpDir)
	if err := language.GenerateFromModel(tmpDir, model, provider, generatedFiles); err != nil {
		return err
	}
	// Collect source root directories needed as include paths for protoc / prost-build
	// compilation. ActiveRoots contains all active root repositories (for example,
	// generating showcase protos requires both "showcase" and "googleapis"). If
	// ActiveRoots is empty, fall back to the codec's default root (codec.RootName).
	var rootPaths []string
	if cfg.Source != nil {
		for _, r := range cfg.Source.ActiveRoots {
			if rootPath := cfg.Source.Root(r); rootPath != "" {
				rootPaths = append(rootPaths, rootPath)
			}
		}
		if len(rootPaths) == 0 {
			if rootPath := cfg.Source.Root(codec.RootName); rootPath != "" {
				rootPaths = append(rootPaths, rootPath)
			}
		}
	}
	return buildRS(ctx, rootPaths, tmpDir, outdir)
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

func buildRS(ctx context.Context, rootPaths []string, tmpDir, outDir string) error {
	var absRoots []string
	for _, r := range rootPaths {
		absRoot, err := filepath.Abs(r)
		if err != nil {
			return err
		}
		absRoots = append(absRoots, absRoot)
	}
	absOutDir, err := filepath.Abs(outDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(absOutDir, 0755); err != nil {
		return err
	}
	env := map[string]string{
		"SOURCE_ROOT": strings.Join(absRoots, string(os.PathListSeparator)),
		"DEST":        absOutDir,
	}
	return command.RunInDirWithEnv(ctx, tmpDir, env, command.Cargo, "build", "--features", "_generate-protos")
}
