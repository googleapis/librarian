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

package generate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/googleapis/generator/internal/gitrepo"
)

func Run(ctx context.Context, arg ...string) error {
	cfg := &config{}
	cfg, err := parseFlags(cfg, arg)
	if err != nil {
		return err
	}
	if _, err := cloneGoogleapis(ctx); err != nil {
		return err
	}
	if _, err := cloneLanguageRepo(ctx, cfg.language); err != nil {
		return err
	}
	return nil
}

const googleapisURL = "https://github.com/googleapis/googleapis"

func cloneGoogleapis(ctx context.Context) (*gitrepo.Repo, error) {
	repoPath := filepath.Join(os.TempDir(), "/generator-googleapis")
	return gitrepo.CloneOrOpen(ctx, repoPath, googleapisURL)
}

func cloneLanguageRepo(ctx context.Context, language string) (*gitrepo.Repo, error) {
	languageRepoURL := fmt.Sprintf("https://github.com/googleapis/google-cloud-%s", language)
	repoPath := filepath.Join(os.TempDir(), fmt.Sprintf("/google-cloud-%s", language))
	return gitrepo.CloneOrOpen(ctx, repoPath, languageRepoURL)
}
