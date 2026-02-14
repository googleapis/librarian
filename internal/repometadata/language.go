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

package repometadata

import (
	"fmt"
	"path"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
)

// goClientDocURL builds the client documentation URL for Go SDK.
func goClientDocURL(library *config.Library, apiPath string) string {
	suffix := fmt.Sprintf("api%s", filepath.Base(apiPath))
	clientDir := clientDirectory(library, apiPath)
	if clientDir != "" {
		suffix = path.Join(clientDir, suffix)
	}
	return fmt.Sprintf("https://cloud.google.com/go/docs/reference/cloud.google.com/go/%s/latest/%s", library.Name, suffix)
}

func clientDirectory(library *config.Library, apiPath string) string {
	if library.Go == nil || library.Go.GoAPIs == nil {
		return ""
	}
	for _, api := range library.Go.GoAPIs {
		if api.Path == apiPath {
			return api.ClientDirectory
		}
	}
	return ""
}
