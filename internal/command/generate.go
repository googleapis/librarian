// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package command

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/googleapis/librarian/internal/container"
)

var CmdGenerate = &Command{
	Name:  "generate",
	Short: "Generate client library code for an API",
	Run: func(ctx context.Context) error {
		if err := validateRequiredFlag("api-path", flagAPIPath); err != nil {
			return err
		}
		if err := validateRequiredFlag("api-root", flagAPIRoot); err != nil {
			return err
		}
		if err := validateLanguage(); err != nil {
			return err
		}

		apiRoot, err := filepath.Abs(flagAPIRoot)
		if err != nil {
			return err
		}

		// tmpRoot is a newly-created working directory under /tmp
		// We do any cloning or copying under there. Currently this is only
		// actually needed in generate if the user hasn't specified an output directory
		// - we could potentially only create it in that case, but always creating it
		// is a more general case.
		tmpRoot, err := createTmpWorkingRoot(time.Now())
		if err != nil {
			return err
		}

		outputDir := filepath.Join(tmpRoot, "output")
		if err := os.Mkdir(outputDir, 0755); err != nil {
			return err
		}
		slog.Info(fmt.Sprintf("Code will be generated in %s", outputDir))

		image := deriveImage(nil)
		if err := container.GenerateRaw(image, apiRoot, outputDir, flagAPIPath); err != nil {
			return err
		}

		if flagBuild {
			if err := container.BuildRaw(image, outputDir, flagAPIPath); err != nil {
				return err
			}
		}
		return nil
	},
}
