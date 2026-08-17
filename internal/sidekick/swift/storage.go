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
	"slices"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/language"
)

type storageAnnotations struct {
	Storage        *api.API
	Control        *api.API
	DefaultHost    string
	ServiceImports []string
	CopyrightYear  string
	PackageVersion string
	BoilerPlate    []string
}

// GenerateStorage generates Swift code for the unified StorageControlClient.
func GenerateStorage(
	ctx context.Context,
	outdir string,
	storageModel *api.API,
	storageModule *config.SwiftModule,
	controlModel *api.API,
	controlModule *config.SwiftModule,
	library *config.Library,
) error {
	provider := func(name string) (string, error) {
		contents, err := templates.ReadFile(name)
		if err != nil {
			return "", err
		}
		return string(contents), nil
	}

	storageCodec, err := newCodec(storageModel, library, storageModule, outdir)
	if err != nil {
		return err
	}
	if err := storageCodec.annotateModel(); err != nil {
		return err
	}

	controlCodec, err := newCodec(controlModel, library, controlModule, outdir)
	if err != nil {
		return err
	}
	if err := controlCodec.annotateModel(); err != nil {
		return err
	}

	defaultHost := "storage.googleapis.com"
	for _, s := range controlModel.Services {
		if s.DefaultHost != "" {
			defaultHost = s.DefaultHost
			break
		}
	}
	if defaultHost == "storage.googleapis.com" {
		for _, s := range storageModel.Services {
			if s.DefaultHost != "" {
				defaultHost = s.DefaultHost
				break
			}
		}
	}

	var serviceImports []string
	importSet := make(map[string]bool)
	for _, model := range []*api.API{storageModel, controlModel} {
		for _, s := range model.Services {
			if sa, ok := s.Codec.(*serviceAnnotations); ok {
				for _, imp := range sa.ServiceImports() {
					if !importSet[imp] && imp != "GoogleCloudGax" && imp != "GoogleCloudAuth" && imp != "Foundation" {
						importSet[imp] = true
						serviceImports = append(serviceImports, imp)
					}
				}
			}
		}
	}
	slices.Sort(serviceImports)

	var boilerPlate []string
	if ma, ok := controlModel.Codec.(*modelAnnotations); ok {
		boilerPlate = ma.BoilerPlate
	}

	var copyrightYear, packageVersion string
	if library != nil {
		copyrightYear = library.CopyrightYear
		packageVersion = library.Version
	}

	mergedModel := &api.API{
		Codec: &storageAnnotations{
			Storage:        storageModel,
			Control:        controlModel,
			DefaultHost:    defaultHost,
			ServiceImports: serviceImports,
			CopyrightYear:  copyrightYear,
			PackageVersion: packageVersion,
			BoilerPlate:    boilerPlate,
		},
	}

	generatedFiles := language.WalkTemplatesDir(templates, "templates/storage")
	generatedFiles = append(generatedFiles, language.GeneratedFile{
		TemplatePath: "templates/common/clients.swift.mustache",
		OutputPath:   "Clients.swift",
	})
	return language.GenerateFromModel(outdir, mergedModel, provider, generatedFiles)
}
