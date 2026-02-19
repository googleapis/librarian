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

package docuploader

import (
	"context"
	"errors"
	"fmt"

	"github.com/googleapis/librarian/internal/command"
)

var (
	errTarFailure          = errors.New("tar reported failure")
	errTarUnexpectedOutput = errors.New("tar reported success but provided unexpected output")
)

// CreateArchive creates a documentation archive suitable for uploading for
// staging. It is expected that the metadata has already been written in
// sourceDir as docs.metadata.json. This is equivalent to running
// `tar czf {targetFile} -C {sourceDir} --xform s:./:: .`
// (The "--xform" part is to avoid ./ being included in all paths in the
// archive.)
func CreateArchive(ctx context.Context, tarExe, sourceDir, targetFile string) error {
	output, err := command.Output(ctx, tarExe, "czf", targetFile, "-C", sourceDir, "--xform", "s:./::", ".")
	if err != nil {
		return fmt.Errorf("failed to create archive %s: %w, %w", targetFile, errTarFailure, err)
	}
	if len(output) > 0 {
		return fmt.Errorf("failed to create archive %s with output %s: %w", targetFile, output, errTarUnexpectedOutput)
	}
	return nil
}
