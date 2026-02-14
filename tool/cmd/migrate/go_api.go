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
					Path:            "google/cloud/bigquery/analyticshub/v1",
					ClientDirectory: "analyticshub",
					ImportPath:      "bigquery/analyticshub",
				},
				{
					Path:            "google/cloud/bigquery/biglake/v1",
					ClientDirectory: "biglake",
					ImportPath:      "bigquery/biglake",
				},
				{
					Path:            "google/cloud/bigquery/biglake/v1alpha1",
					ClientDirectory: "biglake",
					ImportPath:      "bigquery/biglake",
				},
				{
					Path:            "google/cloud/bigquery/connection/v1",
					ClientDirectory: "connection",
					ImportPath:      "bigquery/connection",
				},
				{
					Path:            "google/cloud/bigquery/connection/v1beta1",
					ClientDirectory: "connection",
					ImportPath:      "bigquery/connection",
				},
				{
					Path:            "google/cloud/bigquery/dataexchange/v1beta1",
					ClientDirectory: "dataexchange",
					ImportPath:      "bigquery/dataexchange",
				},
				{
					Path:            "google/cloud/bigquery/datapolicies/v1",
					ClientDirectory: "datapolicies",
					ImportPath:      "bigquery/datapolicies",
				},
				{
					Path:            "google/cloud/bigquery/datapolicies/v1beta1",
					ClientDirectory: "datapolicies",
					ImportPath:      "bigquery/datapolicies",
				},
				{
					Path:            "google/cloud/bigquery/datapolicies/v2",
					ClientDirectory: "datapolicies",
					ImportPath:      "bigquery/datapolicies",
				},
				{
					Path:            "google/cloud/bigquery/datapolicies/v2beta1",
					ClientDirectory: "datapolicies",
					ImportPath:      "bigquery/datapolicies",
				},
				{
					Path:            "google/cloud/bigquery/datatransfer/v1",
					ClientDirectory: "datatransfer",
					ImportPath:      "bigquery/datatransfer",
				},
				{
					Path:            "google/cloud/bigquery/migration/v2",
					ClientDirectory: "migration",
					ImportPath:      "bigquery/migration",
				},
				{
					Path:            "google/cloud/bigquery/migration/v2alpha",
					ClientDirectory: "migration",
					ImportPath:      "bigquery/migration",
				},
				{
					Path:            "google/cloud/bigquery/reservation/v1",
					ClientDirectory: "reservation",
					ImportPath:      "bigquery/reservation",
				},
				{
					Path:            "google/cloud/bigquery/storage/v1",
					ClientDirectory: "storage",
					ImportPath:      "bigquery/storage",
				},
				{
					Path:            "google/cloud/bigquery/storage/v1alpha",
					ClientDirectory: "storage",
					ImportPath:      "bigquery/storage",
				},
				{
					Path:            "google/cloud/bigquery/storage/v1beta",
					ClientDirectory: "storage",
					ImportPath:      "bigquery/storage",
				},
				{
					Path:            "google/cloud/bigquery/storage/v1beta1",
					ClientDirectory: "storage",
					ImportPath:      "bigquery/storage",
				},
				{
					Path:            "google/cloud/bigquery/storage/v1beta2",
					ClientDirectory: "storage",
					ImportPath:      "bigquery/storage",
				},
			},
		},
	}

	libraryOverrides = map[string]*config.Library{
		"ai": {
			ReleaseLevel: "beta",
		},
	}
)
