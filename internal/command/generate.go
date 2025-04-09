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
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/container"
	"github.com/googleapis/librarian/internal/gitrepo"
)

var CmdGenerate = &Command{
	Name:  "generate",
	Short: "Generate client library code for an API",
	flagFunctions: []func(fs *flag.FlagSet){
		addFlagImage,
		addFlagWorkRoot,
		addFlagAPIPath,
		addFlagAPIRoot,
		addFlagLanguage,
		addFlagBuild,
	},
	// Currently we never clone a language repo, and always do raw generation.
	maybeGetLanguageRepo: func(workRoot string) (*gitrepo.Repo, error) {
		return nil, nil
	},
	execute: func(ctx *CommandContext) error {
		if err := validateRequiredFlag("api-path", flagAPIPath); err != nil {
			return err
		}
		if err := validateRequiredFlag("api-root", flagAPIRoot); err != nil {
			return err
		}

		apiRoot, err := filepath.Abs(flagAPIRoot)
		if err != nil {
			return err
		}

		outputDir := filepath.Join(ctx.workRoot, "output")
		if err := os.Mkdir(outputDir, 0755); err != nil {
			return err
		}
		slog.Info(fmt.Sprintf("Code will be generated in %s", outputDir))

		if err := container.GenerateRaw(ctx.image, apiRoot, outputDir, flagAPIPath); err != nil {
			return err
		}

		if flagBuild {
			if err := container.BuildRaw(ctx.image, outputDir, flagAPIPath); err != nil {
				return err
			}
		}
		return nil
	},
}
