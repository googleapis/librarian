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

// channelToServiceConfigOverrides defines API channels whose service configuration does not live
// alongside the API definition directory.
//
// For these entries, the service config must be resolved explicitly rather
// than discovered via directory traversal. All paths are relative to the
// googleapis root.
var channelToServiceConfigOverrides = map[string]string{
	"google/cloud/aiplatform/schema/predict/instance":       "google/cloud/aiplatform/v1/schema/aiplatform_v1.yaml",
	"google/cloud/aiplatform/schema/predict/params":         "google/cloud/aiplatform/v1/schema/aiplatform_v1.yaml",
	"google/cloud/aiplatform/schema/predict/prediction":     "google/cloud/aiplatform/v1/schema/aiplatform_v1.yaml",
	"google/cloud/aiplatform/schema/trainingjob/definition": "google/cloud/aiplatform/v1/schema/aiplatform_v1.yaml",
}
