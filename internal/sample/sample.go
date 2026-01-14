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

// Package sample provides functionality for generating sample values of
// the types contained in the internal package for testing purposes.
package sample

import (
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
)

const (
	// Lib1Name is the name of the first library added to the [Config].
	Lib1Name = "google-cloud-storage"
	// Lib2Name is the name of the second library added to the [Config].
	Lib2Name = "gax-internal"
	// InitialTag is the tag form of [InitialVersion] for use in tests.
	InitialTag = "v1.0.0"
	// InitialVersion is the initial version assigned to libraries in
	// [Config].
	InitialVersion = "1.0.0"
	// NextVersion is the next version typically assigned to libraries
	// starting from [InitialVersion].
	NextVersion = "1.1.0"
	// InitialPreviewTag is the tag form of [InitialPreviewVersion] for use in
	// tests.
	InitialPreviewTag = "v1.1.0-preview.1"
	// InitialPreviewVersion is an initial version that can be assigned to
	// libraries on a preview branch.
	InitialPreviewVersion = "1.1.0-preview.1"
	// NextPreviewPrereleaseVersion is the next prerelease version typically
	// assigned to preview libraries starting from [InitialPreviewVersion].
	NextPreviewPrereleaseVersion = "1.1.0-preview.2"
	// NextPreviewCoreVersion is the next core version typically
	// assigned to preview libraries starting from [InitialPreviewVersion] when
	// the main version has moved on to [NextVersion].
	NextPreviewCoreVersion = "1.2.0-preview.1"
)

var (
	// Lib1Output is the [config.Library] Output path of [Lib1Name] included in
	// [Config].
	Lib1Output = filepath.Join("src", "storage")
	// Lib2Output is the [config.Library] Output path of [Lib2Name] included in
	// [Config].
	Lib2Output = filepath.Join("src", "gax-internal")
)

// Config produces a [config.Config] instance populated with most of the
// properties necessary for testing. It produces a unique instance each time so
// that individual test cases may modify their own instance as needed.
func Config() *config.Config {
	return &config.Config{
		Language: "fake",
		Default:  &config.Default{},
		Release: &config.Release{
			Remote: "origin",
			Branch: "main",
		},
		Sources: &config.Sources{
			Googleapis: &config.Source{
				Commit: "9fcfbea0aa5b50fa22e190faceb073d74504172b",
				SHA256: "81e6057ffd85154af5268c2c3c8f2408745ca0f7fa03d43c68f4847f31eb5f98",
			},
		},
		Libraries: []*config.Library{
			{
				Name:    Lib1Name,
				Version: InitialVersion,
				Output:  Lib1Output,
			},
			{
				Name:    Lib2Name,
				Version: InitialVersion,
				Output:  Lib2Output,
			},
		},
	}
}

// PreviewConfig produces a [config.Config] using the normal [Config] function,
// but modifies the resulting [config.Config] properties to align with that of
// a Preview generation track.
func PreviewConfig() *config.Config {
	c := Config()

	c.Release.Branch = "preview"
	for _, lib := range c.Libraries {
		lib.Version = InitialPreviewVersion
	}

	return c
}
