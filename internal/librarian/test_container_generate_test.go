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
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

func TestNewTestGenerateRunner(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name       string
		cfg        *config.Config
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "valid config",
			cfg: &config.Config{
				Repo:     newTestGitRepo(t).GetDir(),
				WorkRoot: t.TempDir(),
				Image:    "gcr.io/test/test-image",
			},
		},
		{
			name: "missing image",
			cfg: &config.Config{
				Repo:     "https://github.com/googleapis/librarian.git",
				WorkRoot: t.TempDir(),
			},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := newTestGenerateRunner(test.cfg)
			if test.wantErr {
				if err == nil {
					t.Fatalf("newTestGenerateRunner() error = %v, wantErr %v", err, test.wantErr)
				}
				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Fatalf("want error message: %s, got: %s", test.wantErrMsg, err.Error())
				}
				return
			}
		})
	}
}
