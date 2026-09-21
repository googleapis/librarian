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

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian"
	"github.com/googleapis/librarian/internal/librarian/cpp"
)

var (
	errConvertConfig = errors.New("cannot convert configuration")
)

func runCppMigration(ctx context.Context, repoPath string) error {
	data, err := readCppGeneratorConfig(repoPath)
	if err != nil {
		return err
	}
	src, err := fetchSource(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", errFetchSource, err)
	}
	cfg, err := buildCppConfig(data, src)
	if err != nil {
		return err
	}
	// The directory name in Googleapis is present for migration code to look
	// up API details. It shouldn't be persisted.
	if cfg.Sources != nil && cfg.Sources.Googleapis != nil {
		cfg.Sources.Googleapis.Dir = ""
	}
	if err := librarian.RunTidyOnConfig(ctx, repoPath, cfg); err != nil {
		return fmt.Errorf("%w: %w", errTidyFailed, err)
	}
	slog.InfoContext(ctx, "successfully migrated C++ libraries", "count", len(cfg.Libraries))
	return nil
}

func readCppGeneratorConfig(repoPath string) ([]byte, error) {
	path := filepath.Join(repoPath, "generator", "generator_config.textproto")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading generator_config.textproto: %w", err)
	}
	return data, nil
}

func buildCppConfig(data []byte, src *config.Source) (*config.Config, error) {
	cfg, err := cpp.ConvertConfig(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errConvertConfig, err)
	}
	cfg.Repo = "googleapis/google-cloud-cpp"
	cfg.Sources = &config.Sources{
		Googleapis: src,
	}
	return cfg, nil
}
