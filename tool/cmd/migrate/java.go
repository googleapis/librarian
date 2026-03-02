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

package main

import (
	"context"
	"log"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian"
	"github.com/googleapis/librarian/internal/yaml"
)

const (
	generationConfigFileName = "generation_config.yaml"
)

// GAPICConfig represents the GAPIC configuration in generation_config.yaml.
type GAPICConfig struct {
	ProtoPath string `yaml:"proto_path"`
}

// LibraryConfig represents a library entry in generation_config.yaml.
type LibraryConfig struct {
	LibraryName  string        `yaml:"library_name"`
	APIShortName string        `yaml:"api_shortname"`
	GAPICs       []GAPICConfig `yaml:"GAPICs"`
}

// GenerationConfig represents the root of generation_config.yaml.
type GenerationConfig struct {
	Libraries []LibraryConfig `yaml:"libraries"`
}

func runJavaMigration(ctx context.Context, repoPath string) error {
	gen, err := yaml.Read[GenerationConfig](filepath.Join(repoPath, generationConfigFileName))
	if err != nil {
		return err
	}

	cfg := buildConfig(gen)
	if cfg == nil {
		log.Println("Info: No libraries found to migrate.")
		return nil
	}

	if err := librarian.RunTidyOnConfig(ctx, cfg); err != nil {
		return errTidyFailed
	}
	log.Printf("Successfully migrated %d Java libraries", len(cfg.Libraries))
	return nil
}

// buildConfig converts a GenerationConfig to a Librarian Config.
func buildConfig(gen *GenerationConfig) *config.Config {
	var libs []*config.Library
	for _, l := range gen.Libraries {
		name := l.LibraryName
		if name == "" {
			name = l.APIShortName
		}
		var apis []*config.API
		for _, g := range l.GAPICs {
			if g.ProtoPath != "" {
				apis = append(apis, &config.API{Path: g.ProtoPath})
			}
		}
		libs = append(libs, &config.Library{
			Name:   name,
			Output: "java-" + name,
			APIs:   apis,
		})
	}
	if len(libs) == 0 {
		return nil
	}
	return &config.Config{
		Language: "java",
		Default:  &config.Default{},
		Sources: &config.Sources{
			// hardcoded for local testing
			Googleapis: &config.Source{Dir: "../../googleapis"},
		},
		Libraries: libs,
	}
}
