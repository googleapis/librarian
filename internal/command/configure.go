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
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/container"
	"github.com/googleapis/librarian/internal/gitrepo"
	"github.com/googleapis/librarian/internal/statepb"
	"gopkg.in/yaml.v3"
)

type ConfigurationPrContent struct {
	Changes []string
	Errors  []string
}

var CmdConfigure = &Command{
	Name:  "configure",
	Short: "Configure a new API in a given language",
	Run: func(ctx context.Context) error {
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

		outputRoot := filepath.Join(tmpRoot, "output")
		if err := os.Mkdir(outputRoot, 0755); err != nil {
			return err
		}
		slog.Info(fmt.Sprintf("Code will be generated in %s", outputRoot))

		var apiRoot string
		if flagAPIRoot == "" {
			repo, err := cloneGoogleapis(tmpRoot)
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
			languageRepo, err = cloneLanguageRepo(flagLanguage, tmpRoot)
			if err != nil {
				return err
			}
		} else {
			repoRoot, err := filepath.Abs(flagRepoRoot)
			if err != nil {
				return err
			}
			languageRepo, err = gitrepo.Open(repoRoot)
			if err != nil {
				return err
			}
			clean, err := gitrepo.IsClean(languageRepo)
			if err != nil {
				return err
			}
			if !clean {
				return errors.New("language repo must be clean before configuring new APIs")
			}
		}

		state, err := loadState(languageRepo)
		if err != nil {
			return err
		}
		apiPaths, err := findApisToConfigure(apiRoot, state, flagLanguage)
		if err != nil {
			return err
		}

		image := deriveImage(state)

		prContent := ConfigurationPrContent{}
		for _, apiPath := range apiPaths {
			err = configureApi(image, outputRoot, apiRoot, apiPath, languageRepo, &prContent)
			if err != nil {
				return err
			}
		}

		// Need to handle four situations:
		// - No changes, no errors (no PR, process completes normally)
		// - No changes, but there are errors (no PR, log and make the process abort as the only way of drawing attention to the failure)
		// - Some changes, no errors (create PR, process completes normally)
		// - Some changes, some errors (create PR with error messages, process completes normally)
		anyChanges := len(prContent.Changes) > 0
		anyErrors := len(prContent.Errors) > 0

		if !anyChanges && !anyErrors {
			slog.Error("No new APIs to configure.")
			return nil
		} else if !anyChanges && anyErrors {
			slog.Error("No PR to create, but errors were logged. Aborting.")
			return errors.New("errors encountered but no PR to create")
		} else if anyChanges && !anyErrors {
			descriptionText := strings.Join(prContent.Changes, "\n")
			return generateConfigurationPr(ctx, languageRepo, descriptionText, startOfRun)
		} else {
			releasesText := strings.Join(prContent.Changes, "\n")
			errorsText := strings.Join(prContent.Errors, "\n")
			descriptionText := fmt.Sprintf("Configuration Errors:\n==================\n%s\n\n\nChanges Included:\n==================\n%s\n", errorsText, releasesText)
			return generateConfigurationPr(ctx, languageRepo, descriptionText, startOfRun)
		}
	},
}

// Returns a collection of APIs to configure, either from the api-path flag,
// or by examining the service config YAML files to find APIs which have requested libraries,
// but which aren't yet configured.
func findApisToConfigure(apiRoot string, state *statepb.PipelineState, language string) ([]string, error) {
	languageSettingsName := language + "_settings"
	if flagAPIPath != "" {
		return []string{flagAPIPath}, nil
	}
	var apiPaths []string
	err := filepath.WalkDir(apiRoot, func(path string, d fs.DirEntry, err error) error {
		if d.Name() == ".git" {
			return filepath.SkipDir
		}
		if err != nil {
			return err
		}
		// TODO: Validate that we only have a single yaml file per directory.
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".yaml") && !strings.HasSuffix(d.Name(), "gapic.yaml") {
			apiPath, err := filepath.Rel(apiRoot, filepath.Dir(path))
			if err != nil {
				return err
			}
			// If we already generate this library, skip the rest of this directory.
			if findLibrary(state, apiPath) != "" || slices.Contains(state.IgnoredApiPaths, apiPath) {
				return filepath.SkipDir
			}

			generate, err := shouldBeGenerated(path, languageSettingsName)
			if err != nil {
				return err
			}
			if generate {
				apiPaths = append(apiPaths, apiPath)
			}
			// Whether or not we were configured, we can skip the rest of this directory.
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return apiPaths, nil
}

// Loads a service config YAML file from the given path, and looks
// for a set of language settings requesting that the API be generated
// in the given language, with a destination of GITHUB or PACKAGE_MANAGER.
func shouldBeGenerated(serviceYamlPath, languageSettingsName string) (bool, error) {
	data, err := os.ReadFile(serviceYamlPath)
	if err != nil {
		return false, err
	}
	config := make(map[string]interface{})

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return false, err
	}

	t, ok := config["type"].(string)
	if !ok {
		return false, nil
	}
	if t != "google.api.Service" {
		return false, nil
	}

	publishing, ok := config["publishing"].(map[string]interface{})
	if !ok {
		return false, nil
	}
	librarySettings, ok := publishing["library_settings"].([]interface{})
	if !ok {
		return false, nil
	}
	if len(librarySettings) != 1 {
		return false, errors.New("wrong number of library_settings in service config")
	}
	firstSettings, ok := librarySettings[0].(map[string]interface{})
	if !ok {
		return false, nil
	}
	languageSettings, ok := firstSettings[languageSettingsName].(map[string]interface{})
	if !ok {
		return false, nil
	}

	commonSettings, ok := languageSettings["common"].(map[string]interface{})
	if !ok {
		return false, nil
	}
	destinations, ok := commonSettings["destinations"].([]interface{})
	if !ok {
		return false, nil
	}
	for _, destination := range destinations {
		destinationText, ok := destination.(string)
		if ok {
			if destinationText == "GITHUB" || destinationText == "PACKAGE_MANAGER" {
				return true, nil
			}
		}
	}
	return false, nil
}

// Attempts to configure a single API. Steps taken:
// - Run the configure container command
//   - If this fails, indicate that in prDescription and return
//
// - Reformat the state file (which we'd expect to be modified)
// - Check that we now have a library containing the given API (or an ignore entry)
// - Commit the change
// - If we only have an ignore entry, indicate that in prDescription and return
// - Otherwise, try to generate and build the new library
//   - If the generate/build fails, revert the previous commit and indicate that in the prDescription
//   - If the generate/build fails, just reset the working directory (so don't commit the generation) and indicate that in the prDescription
//
// This function only returns an error in the case of non-container failures, which are expected to be fatal.
// If the function returns a non-error, the repo will be clean when the function returns (so can be used for the next step)
func configureApi(image, outputRoot, apiRoot, apiPath string, languageRepo *gitrepo.Repo, prContent *ConfigurationPrContent) error {
	slog.Info(fmt.Sprintf("Configuring %s", apiPath))

	generatorInput := filepath.Join(languageRepo.Dir, "generator-input")
	if err := container.Configure(image, apiRoot, apiPath, generatorInput); err != nil {
		prContent.Errors = append(prContent.Errors, logPartialError(apiPath, err, "configuring"))
		if err := gitrepo.CleanWorkingTree(languageRepo); err != nil {
			return err
		}
		return nil
	}

	// Reload (and then resave, to reformat) the state, so we can find the newly-configured library
	state, err := loadState(languageRepo)
	if err != nil {
		return err
	}
	err = saveState(languageRepo, state)
	if err != nil {
		return err
	}

	// We should now have a library for the given API path, or it should be ignored.
	libraryID := findLibrary(state, apiPath)
	if libraryID == "" {
		if slices.Contains(state.IgnoredApiPaths, apiPath) {
			prContent.Changes = append(prContent.Changes, fmt.Sprintf("Ignoring API path %s", apiPath))
			return nil
		}
		prContent.Errors = append(prContent.Errors, logPartialError(apiPath, err, "finding new library for"))
		if err := gitrepo.CleanWorkingTree(languageRepo); err != nil {
			return err
		}
		return nil
	}

	msg := fmt.Sprintf("feat: Configured library %s for API %s", libraryID, apiPath)
	if err := commitAll(languageRepo, msg); err != nil {
		return err
	}

	// From here on, if we need to report a non-fatal error, we also need to revert the commit we've just created.
	// We generate, clean, copy, build.
	outputDir := filepath.Join(outputRoot, libraryID)
	if err := os.Mkdir(outputDir, 0755); err != nil {
		return err
	}

	if err := container.GenerateLibrary(image, apiRoot, outputDir, generatorInput, libraryID); err != nil {
		prContent.Errors = append(prContent.Errors, logPartialError(libraryID, err, "generating"))
		if err := gitrepo.CleanAndRevertHeadCommit(languageRepo); err != nil {
			return err
		}
		return nil
	}
	if err := container.Clean(image, languageRepo.Dir, libraryID); err != nil {
		prContent.Errors = append(prContent.Errors, logPartialError(libraryID, err, "cleaning"))
		if err := gitrepo.CleanAndRevertHeadCommit(languageRepo); err != nil {
			return err
		}
		return nil
	}
	// If the copy operation fails, it's fine to just fail hard.
	if err := os.CopyFS(languageRepo.Dir, os.DirFS(outputDir)); err != nil {
		return err
	}
	if err := container.BuildLibrary(image, languageRepo.Dir, libraryID); err != nil {
		prContent.Errors = append(prContent.Errors, logPartialError(libraryID, err, "building"))
		if err := gitrepo.CleanAndRevertHeadCommit(languageRepo); err != nil {
			return err
		}
		return nil
	}

	// Success!
	if err := gitrepo.CleanWorkingTree(languageRepo); err != nil {
		return err
	}
	prContent.Changes = append(prContent.Changes, fmt.Sprintf("Configured library %s for API %s", libraryID, apiPath))
	return nil
}

func generateConfigurationPr(ctx context.Context, repo *gitrepo.Repo, prDescription string, startOfRun time.Time) error {
	if !flagPush {
		slog.Info(fmt.Sprintf("Push not specified; would have created configuration PR with the following description:\n%s", prDescription))
		return nil
	}

	title := fmt.Sprintf("feat: API configuration: %s", formatTimestamp(startOfRun))
	_, err := pushAndCreatePullRequest(ctx, repo, time.Now(), title, prDescription)
	return err
}
