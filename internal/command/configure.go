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
	"os"
	"path/filepath"
	"time"

	"github.com/googleapis/librarian/internal/container"
	"github.com/googleapis/librarian/internal/gitrepo"
)

var CmdConfigure = &Command{
	Name:  "configure",
	Short: "Configure a new API in a given language",
	Run: func(ctx context.Context) error {
		if err := validateRequiredFlag("api-path", flagAPIPath); err != nil {
			return err
		}
		if err := validateLanguage(); err != nil {
			return err
		}
		if err := validatePush(); err != nil {
			return err
		}

		startOfRun := time.Now()
		// tmpRoot is a newly-created working directory under /tmp
		// We do any cloning or copying under there. Currently this is only
		// actually needed in generate if the user hasn't specified an output directory
		// - we could potentially only create it in that case, but always creating it
		// is a more general case.
		tmpRoot, err := createTmpWorkingRoot(startOfRun)
		if err != nil {
			return err
		}

		var apiRoot string
		if flagAPIRoot == "" {
			repo, err := cloneGoogleapis(ctx, tmpRoot)
			if err != nil {
				return err
			}
			apiRoot = repo.Dir
		} else {
			// We assume it's okay not to take a defensive copy of apiRoot in the configure command,
			// as "vanilla" configuration/generation shouldn't need to edit any protos. (That's just an escape hatch.)
			apiRoot, err = filepath.Abs(flagAPIRoot)
			if err != nil {
				return err
			}
		}

		var languageRepo *gitrepo.Repo
		if flagRepoRoot == "" {
			languageRepo, err = cloneLanguageRepo(ctx, flagLanguage, tmpRoot)
			if err != nil {
				return err
			}
		} else {
			repoRoot, err := filepath.Abs(flagRepoRoot)
			if err != nil {
				return err
			}
			languageRepo, err = gitrepo.Open(ctx, repoRoot)
			if err != nil {
				return err
			}
		}

		state, err := loadState(languageRepo)
		if err != nil {
			return err
		}

		image := deriveImage(state)

		generatorInput := filepath.Join(languageRepo.Dir, "generator-input")
		if err := container.Configure(ctx, image, apiRoot, flagAPIPath, generatorInput); err != nil {
			return err
		}

		// After configuring, we run quite a lot of the same code as in CmdUpdateApis.Run.
		outputDir := filepath.Join(tmpRoot, "output")
		if err := os.Mkdir(outputDir, 0755); err != nil {
			return err
		}

		// Reload the state, so we can find the newly-configured library
		state, err = loadState(languageRepo)
		if err != nil {
			return err
		}
		libraryID := findLibrary(state, flagAPIPath)

		// Take a defensive copy of the generator input directory from the language repo.
		// Note that we didn't do this earlier, as the container.Configure step is *intended* to modify
		// generator input in the repo. Any changes during generation aren't intended to be persisted though.
		generatorInput = filepath.Join(tmpRoot, "generator-input")
		if err := os.CopyFS(generatorInput, os.DirFS(filepath.Join(languageRepo.Dir, "generator-input"))); err != nil {
			return err
		}

		if err := container.GenerateLibrary(ctx, image, apiRoot, outputDir, generatorInput, libraryID); err != nil {
			return err
		}
		// We don't need to clean the newly-configured API, but we *do* need to clean any non-API-specific files.
		if err := container.Clean(ctx, image, languageRepo.Dir, "none"); err != nil {
			return err
		}
		if err := os.CopyFS(languageRepo.Dir, os.DirFS(outputDir)); err != nil {
			return err
		}
		// TODO: Work out if we actually want to commit the generated code here. Maybe we should just commit the configuration,
		// and let generation happen naturally on the next run.
		// TODO: Update last_generated_commit if we *do* keep generating here.
		msg := fmt.Sprintf("Configured API %s", flagAPIPath) // TODO: Improve info using googleapis commits and version info
		if err := commitAll(ctx, languageRepo, msg); err != nil {
			return err
		}
		// Build the library we've just generated.
		if err := container.BuildLibrary(image, languageRepo.Dir, libraryID); err != nil {
			return err
		}

		return push(ctx, languageRepo, startOfRun, "", "")
	},
}
