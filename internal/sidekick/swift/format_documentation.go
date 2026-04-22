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

package swift

import "strings"

// formatDocumentation converts a documentation string from the source (typically Protobuf
// comments) into a sequence of lines suitable for Swift documentation comments.
//
// Both Swift and the Protobuf comments use markdown, but our markdown includes cross-reference
// links and sometimes needs cleaning up work correctly on a different markdown engine.
func (c *codec) formatDocumentation(doc string) []string {
	if doc == "" {
		return nil
	}
	return strings.Split(doc, "\n")
}
