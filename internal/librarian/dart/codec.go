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
	"errors"
	"fmt"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sidekick/source"
)

var errInvalidSpecificationFormat = errors.New("dart generation requires protobuf specification format")

func toModelConfig(library *config.Library, ch *config.API, sources *source.Sources) (*parser.ModelConfig, error) {
	if library.SpecificationFormat != "" && library.SpecificationFormat != config.SpecProtobuf {
		return nil, fmt.Errorf("%w, got %q", errInvalidSpecificationFormat, library.SpecificationFormat)
	}

	src := addLibraryRoots(library, sources)

	if library.Dart != nil && library.Dart.IncludeList != nil {
		src["include-list"] = strings.Join(library.Dart.IncludeList, ",")
	}
	root := sources.Googleapis
	if ch.Path == "schema/google/showcase/v1beta1" {
		root = sources.Showcase
	}
	svcConfig, err := serviceconfig.Find(root, ch.Path, serviceconfig.LangDart)
	if err != nil {
		return nil, err
	}

	title := svcConfig.Title
	var name string
	if library.Dart != nil {
		name = library.Dart.NameOverride
		if library.Dart.TitleOverride != "" {
			title = library.Dart.TitleOverride
		}
	}

	modelConfig := &parser.ModelConfig{
		SpecificationFormat: config.SpecProtobuf,
		ServiceConfig:       svcConfig.ServiceConfig,
		SpecificationSource: ch.Path,
		Source:              src,
		Codec:               buildCodec(library),
		Override: api.ModelOverride{
			Name:        name,
			Description: library.DescriptionOverride,
			Title:       title,
		},
	}
	return modelConfig, nil
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

// TODO(https://github.com/googleapis/librarian/issues/3863): remove this function once we removed sidekick config.
func addLibraryRoots(library *config.Library, sources *source.Sources) map[string]string {
	src := make(map[string]string)
	if library.Rust == nil {
		library.Rust = &config.RustCrate{}
	}

	if len(library.Roots) == 0 && sources.Googleapis != "" {
		// Default to googleapis if no roots are specified.
		src["googleapis-root"] = sources.Googleapis
		src["roots"] = "googleapis"
	} else {
		src["roots"] = strings.Join(library.Roots, ",")
		rootMap := map[string]struct {
			path string
			key  string
		}{
			"googleapis":   {path: sources.Googleapis, key: "googleapis-root"},
			"discovery":    {path: sources.Discovery, key: "discovery-root"},
			"showcase":     {path: sources.Showcase, key: "showcase-root"},
			"protobuf-src": {path: sources.ProtobufSrc, key: "protobuf-src-root"},
			"conformance":  {path: sources.Conformance, key: "conformance-root"},
		}
		for _, root := range library.Roots {
			if r, ok := rootMap[root]; ok && r.path != "" {
				src[r.key] = r.path
			}
		}
	}

	return src
}
