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

package postprocessing

import (
	"github.com/googleapis/librarian/internal/filesystem"
)

// CopyFile copies a single file from the src path to the dst path.
// It acts as a wrapper around filesystem.CopyFile to provide a unified
// interface for all postprocessing file operations.
func CopyFile(dst, src string) error {
	return filesystem.CopyFile(src, dst)
}
