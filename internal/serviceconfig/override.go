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

package serviceconfig

const (
	titleAppsScriptTypes          = "Google Apps Script Types"
	titleGKEHubTypes              = "GKE Hub Types"
	serviceConfigAIPlatformSchema = "google/cloud/aiplatform/v1/schema/aiplatform_v1.yaml"
)

// Override represents a single API entry with its overrides.
type Override struct {
	ServiceConfig string
	Title         string
}

// Overrides defines the set of API definitions and their overrides.
var Overrides = map[string]Override{
	"google/apps/script/type": {
		Title: titleAppsScriptTypes,
	},
	"google/apps/script/type/calendar": {
		Title: titleAppsScriptTypes,
	},
	"google/apps/script/type/docs": {
		Title: titleAppsScriptTypes,
	},
	"google/apps/script/type/drive": {
		Title: titleAppsScriptTypes,
	},
	"google/apps/script/type/gmail": {
		Title: titleAppsScriptTypes,
	},
	"google/apps/script/type/sheets": {
		Title: titleAppsScriptTypes,
	},
	"google/apps/script/type/slides": {
		Title: titleAppsScriptTypes,
	},
	"google/cloud/aiplatform/v1/schema/predict/instance": {
		ServiceConfig: serviceConfigAIPlatformSchema,
	},
	"google/cloud/aiplatform/v1/schema/predict/params": {
		ServiceConfig: serviceConfigAIPlatformSchema,
	},
	"google/cloud/aiplatform/v1/schema/predict/prediction": {
		ServiceConfig: serviceConfigAIPlatformSchema,
	},
	"google/cloud/aiplatform/v1/schema/trainingjob/definition": {
		ServiceConfig: serviceConfigAIPlatformSchema,
	},
	"google/cloud/gkehub/v1/configmanagement": {
		Title: titleGKEHubTypes,
	},
	"google/cloud/gkehub/v1/multiclusteringress": {
		Title: titleGKEHubTypes,
	},
	"google/cloud/orgpolicy/v1": {
		Title: "Organization Policy Types",
	},
	"google/cloud/oslogin/common": {
		Title: "Cloud OS Login Common Types",
	},
	"google/identity/accesscontextmanager/type": {
		Title: "Access Context Manager Types",
	},
}
