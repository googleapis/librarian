// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package release contains types for language container's release command.
package release

import (
	"github.com/googleapis/librarian/internal/container/java/message"
	"github.com/googleapis/librarian/internal/container/java/pom"
)

// Context has the directory paths for the release-init command.
// https://github.com/googleapis/librarian/blob/main/doc/language-onboarding.md#release-init
type Context struct {
	LibrarianDir string
	RepoDir      string
	OutputDir    string
}

// The Config for the release-init command. This holds the context (the directory paths)
// and the request parsed from the release-init-request.json file.
type Config struct {
	Context *Context
	// This request is parsed from the release-init-request.json file in
	// the LibrarianDir of the context.
	Request *message.ReleaseInitRequest
}

// ReleaseInit handles the release-init command.
func ReleaseInit(request *message.Request, response *message.Response) {
	if err := pom.UpdateVersions(".", request.ReleaseInit.LibraryID, request.ReleaseInit.Version); err != nil {
		response.Error = err.Error()
		return
	}
	response.Success = true
}