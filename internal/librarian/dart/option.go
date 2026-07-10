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
	"fmt"

	"github.com/googleapis/librarian/internal/config"
)

// Option represents the configuration options for generating a Dart library.
type Option struct {
	Name                        string
	Version                     string
	Output                      string
	SpecificationFormat         string
	CopyrightYear               string
	SkipRelease                 bool
	API                         *API
	NameOverride                string
	TitleOverride               string
	IncludeList                 []string
	ExtraImports                string
	APIKeysEnvironmentVariables string
	Dependencies                string
	IssueTrackerURL             string
	LibraryPathOverride         string
	PartFile                    string
	ReadmeAfterTitleText        string
	ReadmeQuickstartText        string
	RepositoryURL               string
	SupportsSSE                 bool
	Packages                    map[string]string
	Prefixes                    map[string]string
	Protos                      map[string]string
}

// API represents the details of a generated API endpoint.
type API struct {
	Path string
}

// NewOption creates a new Option based on the configuration of the library.
func NewOption(library *config.Library) (*Option, error) {
	if err := verify(library); err != nil {
		return nil, err
	}
	res := &Option{
		Name:                library.Name,
		Version:             library.Version,
		Output:              library.Output,
		SpecificationFormat: library.SpecificationFormat,
		CopyrightYear:       library.CopyrightYear,
		SkipRelease:         library.SkipRelease,
	}
	return res, nil
}

func verify(library *config.Library) error {
	if library.SpecificationFormat != "" && library.SpecificationFormat != config.SpecProtobuf {
		return fmt.Errorf("%w, got %q", errInvalidSpecificationFormat, library.SpecificationFormat)
	}
	return nil
}
