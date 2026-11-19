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

package parser

import (
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

// updateAutoPopulatedFields adds auto-populated fields to methods. Fields that
// do not conform to [AIP-4235](https://google.aip.dev/client-libraries/4235)
// are skipped.
//
// The first phases of the parser have no knowledge of the service config
// settings and mark any fields that *might* be auto-populated (having the
// right type and annotations) as `AutoPopulated: true`. This helper applies
// the service configuration settings.
func updateAutoPopulatedFields(serviceConfig *serviceconfig.Service, model *api.API) {
	if serviceConfig == nil {
		return
	}
	methodFields := make(map[string][]string)
	for _, m := range serviceConfig.GetPublishing().GetMethodSettings() {
		if fields := m.GetAutoPopulatedFields(); len(fields) > 0 {
			selector := "." + m.GetSelector()
			methodFields[selector] = append(methodFields[selector], fields...)
		}
	}
	if len(methodFields) > 0 {
		model.ResolveAutoPopulatedFields(methodFields)
	}
}
