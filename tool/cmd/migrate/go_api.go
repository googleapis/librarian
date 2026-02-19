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

package main

import "github.com/googleapis/librarian/internal/config"

var (
	addGoModules = map[string]*RepoConfigModule{
		"ai": {
			APIs: []*RepoConfigAPI{
				{
					Path:            "google/ai/generativelanguage/v1",
					ClientDirectory: "generativelanguage",
					ImportPath:      "ai/generativelanguage",
				},
				{
					Path:            "google/ai/generativelanguage/v1alpha",
					ClientDirectory: "generativelanguage",
					ImportPath:      "ai/generativelanguage",
				},
				{
					Path:            "google/ai/generativelanguage/v1beta",
					ClientDirectory: "generativelanguage",
					ImportPath:      "ai/generativelanguage",
				},
				{
					Path:            "google/ai/generativelanguage/v1beta2",
					ClientDirectory: "generativelanguage",
					ImportPath:      "ai/generativelanguage",
				},
			},
		},
		"bigquery": {
			APIs: []*RepoConfigAPI{
				{
					Path:               "google/cloud/bigquery/analyticshub/v1",
					NoRESTNumericEnums: true,
				},
				{
					Path:               "google/cloud/bigquery/dataexchange/v1beta1",
					NoRESTNumericEnums: true,
				},
				{
					Path:               "google/cloud/bigquery/datapolicies/v1beta1",
					NoRESTNumericEnums: true,
				},
				{
					Path:               "google/cloud/bigquery/migration/v2",
					NoRESTNumericEnums: true,
				},
				{
					Path:               "google/cloud/bigquery/migration/v2alpha",
					NoRESTNumericEnums: true,
				},
				{
					Path:               "google/cloud/bigquery/storage/v1",
					NoRESTNumericEnums: true,
				},
				{
					Path:               "google/cloud/bigquery/storage/v1beta1",
					NoRESTNumericEnums: true,
				},
				{
					Path:               "google/cloud/bigquery/storage/v1beta2",
					NoRESTNumericEnums: true,
				},
			},
		},
	}

	addKeep = map[string][]string{
		"bigquery": {
			"README.md",
		},
	}

	libraryOverrides = map[string]*config.Library{
		"ai": {
			ReleaseLevel: "beta",
		},
	}

	nestedMods = map[string]string{
		"bigquery": "v2",
	}
)
