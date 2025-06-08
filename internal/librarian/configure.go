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

package librarian

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/container"
	"github.com/googleapis/librarian/internal/gitrepo"
	"github.com/googleapis/librarian/internal/statepb"
	"gopkg.in/yaml.v3"
)

var CmdConfigure = &cli.Command{
	Name:  "configure",
	Short: "Set up a new API for a language.",
	Usage: "TODO(https://github.com/googleapis/librarian/issues/237): add documentation",
	Long:  "TODO(https://github.com/googleapis/librarian/issues/237): add documentation",
	Run:   runConfigure,
}

func init() {
	CmdConfigure.SetFlags([]func(fs *flag.FlagSet){
		addFlagImage,
		addFlagWorkRoot,
		addFlagAPIPath,
		addFlagAPIRoot,
		addFlagGitUserEmail,
		addFlagGitUserName,
		addFlagLanguage,
		addFlagPush,
		addFlagRepoRoot,
		addFlagRepoUrl,
		addFlagSecretsProject,
	})
}

func runConfigure(ctx context.Context) error {
	state, err := createCommandStateForLanguage(ctx)
	if err != nil {
		return err
	}
	return executeConfigure(state)
}

func executeConfigure(state *commandState) error {
	if err := validatePush(); err != nil {
		return err
	}

	outputRoot := filepath.Join(state.workRoot, "output")
	if err := os.Mkdir(outputRoot, 0755); err != nil {
		return err
	}
	slog.Info(fmt.Sprintf("Code will be generated in %s", outputRoot))

	var apiRoot string
	if flagAPIRoot == "" {
		repo, err := cloneGoogleapis(state.workRoot)
		if err != nil {
			return err
		}
		apiRoot = repo.Dir
	} else {
		// We assume it's okay not to take a defensive copy of apiRoot in the configure command,
		// as "vanilla" configuration/generation shouldn't need to edit any protos. (That's just an escape hatch.)
		absRoot, err := filepath.Abs(flagAPIRoot)
		if err != nil {
			return err
		}
		apiRoot = absRoot
	}
	apiPaths, err := findApisToConfigure(apiRoot, state.pipelineState, flagLanguage)
	if err != nil {
		return err
	}

	prContent := PullRequestContent{}
	for _, apiPath := range apiPaths {
		err = configureApi(state, outputRoot, apiRoot, apiPath, &prContent)
		if err != nil {
			return err
		}
	}

	_, err = createPullRequest(state, &prContent, "feat: API configuration", "", "config")
	return err
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
			if findLibraryIDByApiPath(state, apiPath) != "" || slices.Contains(state.IgnoredApiPaths, apiPath) {
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
func configureApi(state *commandState, outputRoot, apiRoot, apiPath string, prContent *PullRequestContent) error {
	containerConfig := state.containerConfig
	languageRepo := state.languageRepo

	slog.Info(fmt.Sprintf("Configuring %s", apiPath))

	generatorInput := filepath.Join(languageRepo.Dir, "generator-input")
	if err := container.Configure(containerConfig, apiRoot, apiPath, generatorInput); err != nil {
		addErrorToPullRequest(prContent, apiPath, err, "configuring")
		if err := gitrepo.CleanWorkingTree(languageRepo); err != nil {
			return err
		}
		return nil
	}

	// Reload (and then resave, to reformat) the ps, so we can find the newly-configured library
	ps, err := loadRepoPipelineState(languageRepo)
	if err != nil {
		return err
	}
	state.pipelineState = ps
	err = savePipelineState(state)
	if err != nil {
		return err
	}

	// We should now have a library for the given API path, or it should be ignored.
	libraryID := findLibraryIDByApiPath(ps, apiPath)
	if libraryID == "" {
		// If it's newly-ignored, just commit the state change. This is still a "success" case.
		if slices.Contains(ps.IgnoredApiPaths, apiPath) {
			msg := fmt.Sprintf("feat: Added ignore entry for API %s", apiPath)
			if err := commitAll(languageRepo, msg); err != nil {
				return err
			}
			addSuccessToPullRequest(prContent, fmt.Sprintf("Ignored API %s", apiPath))
			return nil
		}
		addErrorToPullRequest(prContent, apiPath, err, "finding new library for")
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

	if err := container.GenerateLibrary(containerConfig, apiRoot, outputDir, generatorInput, libraryID); err != nil {
		prContent.Errors = append(prContent.Errors, logPartialError(libraryID, err, "generating"))
		if err := gitrepo.CleanAndRevertHeadCommit(languageRepo); err != nil {
			return err
		}
		return nil
	}
	if err := container.Clean(containerConfig, languageRepo.Dir, libraryID); err != nil {
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
	if err := container.BuildLibrary(containerConfig, languageRepo.Dir, libraryID); err != nil {
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
	addSuccessToPullRequest(prContent, fmt.Sprintf("Configured library %s for API %s", libraryID, apiPath))
	return nil
}
