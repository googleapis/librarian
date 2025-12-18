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

package utils

import (
	"github.com/googleapis/librarian/internal/sidekick/api"
)

// GetGcloudType maps a proto data type to its corresponding gcloud type.
func GetGcloudType(t api.Typez) string {
	switch t {
	case api.STRING_TYPE:
		return "" // Default is string
	case api.INT32_TYPE, api.INT64_TYPE, api.UINT32_TYPE, api.UINT64_TYPE:
		return "long"
	case api.BOOL_TYPE:
		return "boolean"
	case api.FLOAT_TYPE, api.DOUBLE_TYPE:
		return "float"
	default:
		return ""
	}
}


