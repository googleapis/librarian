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

package dart

import (
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian/common"
	"github.com/googleapis/librarian/internal/serviceconfig"
	sidekickconfig "github.com/googleapis/librarian/internal/sidekick/config"
)

func toSidekickConfig(library *config.Library, ch *config.API, sources *common.Sources) (*sidekickconfig.Config, error) {
	source := common.AddLibraryRoots(library, sources)

	if library.DescriptionOverride != "" {
		source["description-override"] = library.DescriptionOverride
	}
	if library.Dart != nil && library.Dart.NameOverride != "" {
		source["name-override"] = library.Dart.NameOverride
	}
	if library.Dart != nil && library.Dart.TitleOverride != "" {
		source["title-override"] = library.Dart.TitleOverride
	}

	root := sources.Googleapis
	if ch.Path == "schema/google/showcase/v1beta1" {
		root = sources.Showcase
	}
	api, err := serviceconfig.Find(root, ch.Path)
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
	if dart.NotForPublication != "" {
		codec["not-for-publication"] = dart.NotForPublication
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
