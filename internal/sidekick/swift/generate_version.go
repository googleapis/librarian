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

package swift

import (
	"context"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/license"
	"github.com/googleapis/librarian/internal/sidekick/language"
)

type versionView struct {
	library *config.Library
}

func (v *versionView) CopyrightYear() string {
	return v.library.CopyrightYear
}

func (v *versionView) Version() string {
	return v.library.Version
}

func (v *versionView) BoilerPlate() []string {
	return license.HeaderBulk()
}

// GenerateVersion generates a version file for core and veneers.
//
// Swift does not provide programmatic access to the version of a package. The
// version information is in `git`, not in any manifest file. In GAPICs and
// core libraries we need the version to populate telemetry headers.
//
// This function generates a small "version file" based on the information in
// librarian.
func GenerateVersion(ctx context.Context, outDir string, library *config.Library) error {
	return parsedTemplates.GenerateElement(outDir, &versionView{library: library}, language.GeneratedFile{
		TemplatePath: "templates/version/version.swift.gotmpl",
		OutputPath:   "PackageVersion.swift",
	})
}
