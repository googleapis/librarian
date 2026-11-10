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

package ruby

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"golang.org/x/sync/errgroup"
)

const (
	maxFilesPerFormatBatch = 200
	generatorGemName       = "gapic-generator-cloud"
	rubocopConfigFileName  = "cloud-rubocop.yml"
)

// ErrRubocopConfigNotFound indicates that the cloud-rubocop.yml configuration file could not be found.
var ErrRubocopConfigNotFound = errors.New("cloud-rubocop.yml not found in gapic-generator-cloud gem")

// Format formats generated Ruby libraries using rubocop in batches.
func Format(ctx context.Context, tools *config.Tools, libraries ...*config.Library) error {
	files, err := collectAllRubyFiles(libraries)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}
	env, err := toolsEnv()
	if err != nil {
		return err
	}
	configPath, err := rubocopConfigPath(tools)
	if err != nil {
		return err
	}
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(runtime.GOMAXPROCS(0))
	for chunk := range slices.Chunk(files, maxFilesPerFormatBatch) {
		group.Go(func() error {
			return command.RunWithEnv(groupCtx, env, "rubocop", append([]string{"--cache", "false", "-a", "--fail-level", "E", "-c", configPath}, chunk...)...)
		})
	}
	return group.Wait()
}

func collectAllRubyFiles(libraries []*config.Library) ([]string, error) {
	var allFiles []string
	for _, library := range libraries {
		files, err := collectRubyFiles(library)
		if err != nil {
			return nil, fmt.Errorf("collect ruby files in %q: %w", library.Name, err)
		}
		allFiles = append(allFiles, files...)
	}
	return allFiles, nil
}

func rubocopConfigPath(tools *config.Tools) (string, error) {
	if tools == nil {
		return "", ErrRubocopConfigNotFound
	}
	index := slices.IndexFunc(tools.Gem, func(gemTool *config.GemTool) bool {
		return gemTool.Name == generatorGemName && gemTool.Version != ""
	})
	if index == -1 {
		return "", ErrRubocopConfigNotFound
	}
	installDir, err := InstallDir()
	if err != nil {
		return "", err
	}
	configPath := filepath.Join(installDir, "gems", fmt.Sprintf("%s-%s", generatorGemName, tools.Gem[index].Version), rubocopConfigFileName)
	if _, err := os.Stat(configPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", ErrRubocopConfigNotFound
		}
		return "", err
	}
	return configPath, nil
}

func collectRubyFiles(library *config.Library) ([]string, error) {
	keepSet := buildKeepSet(library.Name, library.Keep)
	var files []string
	err := filepath.WalkDir(library.Output, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		relPath, err := filepath.Rel(library.Output, path)
		if err != nil {
			return err
		}
		relSlash := filepath.ToSlash(relPath)
		if isFormattableRubyFile(relSlash) && !isKept(relSlash, keepSet) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func isFormattableRubyFile(relSlash string) bool {
	switch relSlash {
	case ".toys.rb", "Gemfile", "snippets/Gemfile":
		return true
	}
	dir, _, _ := strings.Cut(relSlash, "/")
	switch dir {
	case "lib", "proto_docs", "snippets":
		return strings.HasSuffix(relSlash, ".rb") && !strings.HasSuffix(relSlash, "_pb.rb")
	default:
		return false
	}
}
