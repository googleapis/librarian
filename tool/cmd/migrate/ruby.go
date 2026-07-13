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
	"fmt"
	"log"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian"
)

func runRubyMigration(ctx context.Context, repoPath string) error {
	src, err := fetchSource(ctx)
	if err != nil {
		return errFetchSource
	}
	cfg := &config.Config{
		Language: config.LanguageRuby,
		Sources: &config.Sources{
			Googleapis: src,
		},
		Tools: &config.Tools{
			Gem: []*config.GemTool{
				{
					Name:    "gapic-generator",
					Version: "0.49.0",
				},
				{
					Name:    "grpc",
					Version: "1.78.1",
				},
			},
			Protoc: &config.Protoc{
				Version: "33.2",
				SHA256:  "b24b53f87c151bfd48b112fe4c3a6e6574e5198874f38036aff41df3456b8caf",
			},
		},
	}
	// The directory name in Googleapis is present for migration code to look
	// up API details. It shouldn't be persisted.
	cfg.Sources.Googleapis.Dir = ""
	if err := librarian.RunTidyOnConfig(ctx, repoPath, cfg); err != nil {
		return fmt.Errorf("%w: %w", errTidyFailed, err)
	}
	log.Printf("Successfully migrated Ruby libraries configuration skeleton")
	return nil
}
