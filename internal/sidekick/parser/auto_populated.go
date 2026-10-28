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

// findAutoPopulatedFields returns the auto-populated fields for a method with
// ID `methodID` and input message `message`. Fields that do not conform to
// [AIP-4235](https://google.aip.dev/client-libraries/4235) are skipped.
//
// The first phases of the parser have no knowledge of the service config
// settings and mark any fields that *might* be auto-populated (having the
// right type and annotations) as `AutoPopulated: true`. This helper applies
// the service configuration settings.
func findAutoPopulatedFields(serviceConfig *serviceconfig.Service, methodID string, message *api.Message) []*api.Field {
	if serviceConfig == nil || message == nil {
		return nil
	}
	var result []*api.Field
	for _, m := range serviceConfig.GetPublishing().GetMethodSettings() {
		if "."+m.GetSelector() != methodID {
			continue
		}
		for _, name := range m.GetAutoPopulatedFields() {
			for _, field := range message.Fields {
				if field.Name != name {
					continue
				}
				if field.AutoPopulated {
					result = append(result, field)
				}
				break
			}
		}
	}
	return result
}
