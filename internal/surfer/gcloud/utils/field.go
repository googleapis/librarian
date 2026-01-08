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

// Package utils provides utility functions for the gcloud generator.
package utils

import (
	"fmt"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

// GetGcloudType maps an API field type to the corresponding gcloud argparse type.
func GetGcloudType(t api.Typez) string {
	switch t {
	case api.DOUBLE_TYPE, api.FLOAT_TYPE:
		return "float"
	case api.INT64_TYPE, api.UINT64_TYPE, api.FIXED64_TYPE, api.SFIXED64_TYPE, api.SINT64_TYPE:
		return "long"
	case api.INT32_TYPE, api.UINT32_TYPE, api.FIXED32_TYPE, api.SFIXED32_TYPE, api.SINT32_TYPE:
		return "int"
	case api.BOOL_TYPE:
		return "bool"
	case api.STRING_TYPE, api.ENUM_TYPE:
		return "str"
	case api.BYTES_TYPE:
		return "bytes"
	case api.MESSAGE_TYPE, api.GROUP_TYPE:
		return "arg_object"
	default:
		panic(fmt.Sprintf("unsupported API type: %v", t))
	}
}
