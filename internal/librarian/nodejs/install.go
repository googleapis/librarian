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

package nodejs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/command"
)

// TODO(https://github.com/googleapis/librarian/issues/4848): read tool
// versions from google-cloud-node/librarian.yaml instead of hardcoding them.
const (
	// TODO(https://github.com/googleapis/librarian/issues/4904): use
	// google-cloud-node instead of google-cloud-node-core.
	gapicGeneratorTypescript     = "gapic-generator-typescript"
	gapicGeneratorTypescriptRepo = "https://github.com/googleapis/google-cloud-node-core.git"
	gapicNodeProcessingPkg       = "gapic-node-processing@0.1.7"
	gapicToolsPkg                = "gapic-tools@1.0.5"
	synthtoolPkg                 = "gcp-synthtool@git+https://github.com/googleapis/synthtool@5aa438a342707842d11fbbb302c6277fbf9e4655"
)

// Install installs Node.js tool dependencies.
func Install(ctx context.Context) error {
	if err := installGapicGeneratorTypescript(ctx); err != nil {
		return err
	}
	if err := command.RunStreaming(ctx, "npm", "install", "-g", gapicNodeProcessingPkg, gapicToolsPkg); err != nil {
		return err
	}
	return command.RunStreaming(ctx, "pip", "install", synthtoolPkg)
}

func installGapicGeneratorTypescript(ctx context.Context) error {
	dir, err := os.MkdirTemp("", fmt.Sprintf("%s-*", gapicGeneratorTypescript))
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	if err := command.RunStreaming(ctx, "git", "clone", "--depth", "1", gapicGeneratorTypescriptRepo, dir); err != nil {
		return err
	}
	genDir := filepath.Join(dir, "generator", gapicGeneratorTypescript)
	if err := command.RunInDir(ctx, genDir, "npm", "install"); err != nil {
		return err
	}
	if err := command.RunInDir(ctx, genDir, "npm", "run", "compile"); err != nil {
		return err
	}
	return command.RunInDir(ctx, genDir, "npm", "link")
}
