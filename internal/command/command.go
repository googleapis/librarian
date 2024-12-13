// Copyright 2024 Google LLC
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
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/googleapis/generator/internal/container"
)

type Command struct {
	Name  string
	Short string
	Run   func(ctx context.Context) error

	flags *flag.FlagSet
}

func (c *Command) Parse(args []string) error {
	return c.flags.Parse(args)
}

func Lookup(name string) (*Command, error) {
	var cmd *Command
	for _, sub := range Commands {
		if sub.Name == name {
			cmd = sub
		}
	}
	if cmd == nil {
		return nil, fmt.Errorf("invalid command: %q", name)
	}
	return cmd, nil
}

var CmdConfigure = &Command{
	Name:  "configure",
	Short: "Configure a new API in a given language",
	Run: func(ctx context.Context) error {
		if flagAPIRoot == "" {
			return fmt.Errorf("-api-root is not provided")
		}
		if flagAPIPath == "" {
			return fmt.Errorf("-api-path is not provided")
		}
		if !supportedLanguages[flagLanguage] {
			return fmt.Errorf("invalid -language flag specified: %q", flagLanguage)
		}
		if flagPush && flagGitHubToken == "" {
			return fmt.Errorf("-github-token must be provided if -push is set to true")
		}

		// We assume it's okay not to take a defensive copy of apiRoot in the configure command,
		// as "vanilla" generation shouldn't need to edit any protos. (That's just an escape hatch.)
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

		image := deriveImage()
		languageRepo, err := cloneLanguageRepo(ctx, flagLanguage, tmpRoot)
		if err != nil {
			return err
		}

		generatorInput := filepath.Join(languageRepo.Dir, "generator-input")
		if err := container.Configure(ctx, image, apiRoot, flagAPIPath, generatorInput); err != nil {
			return err
		}

		// After configuring, we run quite a lot of the same code as in CmdUpdateRepo.Run.
		outputDir := filepath.Join(tmpRoot, "output")
		if err := os.Mkdir(outputDir, 0755); err != nil {
			return err
		}

		// Take a defensive copy of the generator input directory from the language repo.
		// Note that we didn't do this earlier, as the container.Configure step is *intended* to modify
		// generator input in the repo. Any changes during generation aren't intended to be persisted though.
		generatorInput = filepath.Join(tmpRoot, "generator-input")
		if err := os.CopyFS(generatorInput, os.DirFS(filepath.Join(languageRepo.Dir, "generator-input"))); err != nil {
			return err
		}

		if err := container.Generate(ctx, image, apiRoot, outputDir, generatorInput, flagAPIPath); err != nil {
			return err
		}
		// No need to clean here, as configure should fail for an existing API.
		if err := os.CopyFS(languageRepo.Dir, os.DirFS(outputDir)); err != nil {
			return err
		}
		if err := container.Build(ctx, image, "repo-root", languageRepo.Dir, flagAPIPath); err != nil {
			return err
		}

		if err := commit(); err != nil {
			return err
		}
		return push()
	},
}

var CmdGenerate = &Command{
	Name:  "generate",
	Short: "Generate a new client library",
	Run: func(ctx context.Context) error {
		if flagAPIPath == "" {
			return fmt.Errorf("-api-path is not provided")
		}
		if !supportedLanguages[flagLanguage] {
			return fmt.Errorf("invalid -language flag specified: %q", flagLanguage)
		}
		if flagAPIRoot == "" {
			return fmt.Errorf("-api-root is not provided")
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

		var outputDir string
		if flagOutput == "" {
			outputDir = filepath.Join(tmpRoot, "output")
			if err := os.Mkdir(outputDir, 0755); err != nil {
				return err
			}
			slog.Info(fmt.Sprintf("No output directory specified. Defaulting to %s", outputDir))
		} else {
			outputDir, err = filepath.Abs(flagOutput)
			if err != nil {
				return err
			}
		}

		image := deriveImage()
		// The final empty string argument is for generator input - we don't have any
		if err := container.Generate(ctx, image, apiRoot, outputDir, "", flagAPIPath); err != nil {
			return err
		}

		if flagBuild {
			if err := container.Build(ctx, image, "generator-output", outputDir, flagAPIPath); err != nil {
				return err
			}
		}
		return nil
	},
}

var CmdUpdateRepo = &Command{
	Name:  "update-repo",
	Short: "Configure a new API in a given language",
	Run: func(ctx context.Context) error {

		if !supportedLanguages[flagLanguage] {
			return fmt.Errorf("invalid -language flag specified: %q", flagLanguage)
		}

		// tmpRoot is a newly-created working directory under /tmp
		// We do any cloning or copying under there.
		tmpRoot, err := createTmpWorkingRoot(time.Now())
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
			// Take a defensive copy of googleapis; ideally we'd omit
			// the .git directory here, but this is at least simple.
			apiRoot = filepath.Join(tmpRoot, "googleapis")
			slog.Info(fmt.Sprintf("Copying %s to %s", flagAPIRoot, apiRoot))
			os.CopyFS(apiRoot, os.DirFS(flagAPIRoot))
		}

		var outputDir string
		if flagOutput == "" {
			outputDir = filepath.Join(tmpRoot, "output")
			if err := os.Mkdir(outputDir, 0755); err != nil {
				return err
			}
			slog.Info(fmt.Sprintf("No output directory specified. Defaulting to %s", outputDir))
		} else {
			outputDir, err = filepath.Abs(flagOutput)
			if err != nil {
				return err
			}
		}

		languageRepo, err := cloneLanguageRepo(ctx, flagLanguage, tmpRoot)
		if err != nil {
			return err
		}

		image := deriveImage()

		// Take a defensive copy of the generator input directory from the language repo.
		generatorInput := filepath.Join(tmpRoot, "generator-input")
		if err := os.CopyFS(generatorInput, os.DirFS(filepath.Join(languageRepo.Dir, "generator-input"))); err != nil {
			return err
		}

		if err := container.Generate(ctx, image, apiRoot, outputDir, generatorInput, flagAPIPath); err != nil {
			return err
		}
		if err := container.Clean(ctx, image, languageRepo.Dir, flagAPIPath); err != nil {
			return err
		}
		if err := os.CopyFS(languageRepo.Dir, os.DirFS(outputDir)); err != nil {
			return err
		}
		if err := container.Build(ctx, image, "repo-root", languageRepo.Dir, flagAPIPath); err != nil {
			return err
		}
		if err := commit(); err != nil {
			return err
		}
		return push()
	},
}

func deriveImage() string {
	if flagImage != "" {
		return flagImage
	} else {
		return fmt.Sprintf("google-cloud-%s-generator", flagLanguage)
	}
}

func createTmpWorkingRoot(t time.Time) (string, error) {
	const yyyyMMddHHmmss = "20060102T150405" // Expected format by time library

	path := filepath.Join(os.TempDir(), fmt.Sprintf("generator-%s", t.Format(yyyyMMddHHmmss)))

	_, err := os.Stat(path)
	switch {
	case os.IsNotExist(err):
		if err := os.Mkdir(path, 0755); err != nil {
			return "", fmt.Errorf("unable to create temporary working directory '%s': %w", path, err)
		}
	case err == nil:
		return "", fmt.Errorf("temporary working directory already exists: %s", path)
	default:
		return "", fmt.Errorf("unable to check directory '%s': %w", path, err)
	}

	slog.Info(fmt.Sprintf("Temporary working directory: %s", path))
	return path, nil
}

func commit() error {
	return fmt.Errorf("commit is not implemented")
}

func push() error {
	return fmt.Errorf("push is not implemented")
}

var Commands = []*Command{
	CmdConfigure,
	CmdGenerate,
	CmdUpdateRepo,
}

func init() {
	for _, c := range Commands {
		c.flags = flag.NewFlagSet(c.Name, flag.ContinueOnError)
		c.flags.Usage = constructUsage(c.flags, c.Name)
	}

	fs := CmdConfigure.flags
	for _, fn := range []func(fs *flag.FlagSet){
		addFlagImage,
		addFlagAPIPath,
		addFlagAPIRoot,
		addFlagLanguage,
		addFlagPush,
		addFlagGitHubToken,
	} {
		fn(fs)
	}

	fs = CmdGenerate.flags
	for _, fn := range []func(fs *flag.FlagSet){
		addFlagImage,
		addFlagAPIPath,
		addFlagAPIRoot,
		addFlagLanguage,
		addFlagOutput,
		addFlagBuild,
	} {
		fn(fs)
	}

	fs = CmdUpdateRepo.flags
	for _, fn := range []func(fs *flag.FlagSet){
		addFlagImage,
		addFlagAPIPath,
		addFlagAPIRoot,
		addFlagBranch,
		addFlagGitHubToken,
		addFlagLanguage,
		addFlagOutput,
		addFlagPush,
	} {
		fn(fs)
	}
}

func constructUsage(fs *flag.FlagSet, name string) func() {
	output := fmt.Sprintf("Usage:\n\n  generator %s [arguments]\n", name)
	output += "\nFlags:\n\n"
	return func() {
		fmt.Fprint(fs.Output(), output)
		fs.PrintDefaults()
		fmt.Fprintf(fs.Output(), "\n\n")
	}
}
