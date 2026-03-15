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

var (
	// keep maps a Go library to a list of files that should be preserved during
	// generation. This is a hardcoded list to handle special cases during legacy
	// migration where the legacy librarian handled file preservation differently.
	keep = map[string][]string{
		"auth":              {"internal/version.go", "README.md"},
		"auth/oauth2adapt":  {"internal/version.go"},
		"batch":             {"apiv1/iam_policy_client.go"},
		"bigquery":          {"README.md"},
		"compute/metadata":  {"internal/version.go", "README.md"},
		"containeranalysis": {"apiv1beta1/grafeas/grafeaspb/grafeas.pb.go"},
		"datacatalog":       {"apiv1/iam_policy_client.go"},
		"datastream":        {"apiv1/iam_policy_client.go"},
		"grafeas":           {"internal/version.go", "README.md"},
		"profiler":          {"internal/version.go", "README.md"},
		"pubsub":            {"internal/version.go", "README.md"},
		"run":               {"apiv2/locations_client.go"},
		"spanner":           {"README.md"},
		"storage":           {"README.md"},
		"vertexai":          {"internal/version.go", "README.md"},
		"vmmigration":       {"apiv1/iam_policy_client.go"},
	}
	// nestedModules maps specific Go libraries to their nested module path.
	// This is a hardcoded list to handle special cases during legacy migration
	// where this information is not available in the source configuration.
	nestedModules = map[string]string{
		"bigquery": "v2",
		"compute":  "metadata",
		"iam":      "admin",
		"logging":  "logadmin",
		"pubsub":   "v2",
	}
)
