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

// Package dart provides functionality for generating and releasing Dart client
// libraries.
package dart

import (
	"context"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	sidekickconfig "github.com/googleapis/librarian/internal/sidekick/config"
	sidekickdart "github.com/googleapis/librarian/internal/sidekick/dart"
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

// Generate generates a Dart client library.
func Generate(ctx context.Context, library *config.Library, googleapisDir string) error {
	sidekickConfig, err := toSidekickConfig(library, library.APIs[0], googleapisDir)
	if err != nil {
		return err
	}
	model, err := parser.CreateModel(sidekickConfig)
	if err != nil {
		return err
	}
	if err := sidekickdart.Generate(ctx, model, library.Output, sidekickConfig); err != nil {
		return err
	}
	return nil
}

// Format formats a generated Dart library.
func Format(ctx context.Context, library *config.Library) error {
	if err := command.Run(ctx, "dart", "format", library.Output); err != nil {
		return err
	}
	return nil
}

func toSidekickConfig(library *config.Library, ch *config.API, googleapisDir string) (*sidekickconfig.Config, error) {
	source := map[string]string{
		"googleapis-root": googleapisDir,
	}

	if library.DescriptionOverride != "" {
		source["description-override"] = library.DescriptionOverride
	}
	if library.Dart != nil && library.Dart.NameOverride != "" {
		source["name-override"] = library.Dart.NameOverride
	}
	if library.Dart != nil && library.Dart.TitleOverride != "" {
		source["title-override"] = library.Dart.TitleOverride
	}
	if library.Dart != nil && len(library.Dart.IncludeList) > 0 {
		source["include-list"] = strings.Join(library.Dart.IncludeList, ",")
	}

	api, err := serviceconfig.Find(googleapisDir, ch.Path)
	if err != nil {
		return nil, err
	}

	sidekickCfg := &sidekickconfig.Config{
		General: sidekickconfig.GeneralConfig{
			Language:            "dart",
			SpecificationFormat: "protobuf",
			ServiceConfig:       api.ServiceConfig,
			SpecificationSource: ch.Path,
		},
		Source: source,
		Codec:  buildCodec(library),
	}
	return sidekickCfg, nil
}

func buildCodec(library *config.Library) map[string]string {
	codec := make(map[string]string)
	if library.CopyrightYear != "" {
		codec["copyright-year"] = library.CopyrightYear
	}
	if library.Version != "" {
		codec["version"] = library.Version
	}
	if library.SkipPublish {
		codec["not-for-publication"] = "true"
	}
	if library.Dart == nil {
		return codec
	}

	dart := library.Dart
	if dart.APIKeysEnvironmentVariables != "" {
		codec["api-keys-environment-variables"] = dart.APIKeysEnvironmentVariables
	}
	if dart.Dependencies != "" {
		codec["dependencies"] = dart.Dependencies
	}
	if dart.DevDependencies != "" {
		codec["dev-dependencies"] = dart.DevDependencies
	}
	if dart.ExtraImports != "" {
		codec["extra-imports"] = dart.ExtraImports
	}
	if dart.IssueTrackerURL != "" {
		codec["issue-tracker-url"] = dart.IssueTrackerURL
	}
	if dart.LibraryPathOverride != "" {
		codec["library-path-override"] = dart.LibraryPathOverride
	}
	if dart.PartFile != "" {
		codec["part-file"] = dart.PartFile
	}
	if dart.ReadmeAfterTitleText != "" {
		codec["readme-after-title-text"] = dart.ReadmeAfterTitleText
	}
	if dart.ReadmeQuickstartText != "" {
		codec["readme-quickstart-text"] = dart.ReadmeQuickstartText
	}
	if dart.RepositoryURL != "" {
		codec["repository-url"] = dart.RepositoryURL
	}
	for key, value := range dart.Packages {
		codec[key] = value
	}
	for key, value := range dart.Prefixes {
		codec[key] = value
	}
	for key, value := range dart.Protos {
		codec[key] = value
	}
	return codec
}
