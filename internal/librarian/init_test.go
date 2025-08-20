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
	"os"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

func TestNewInitRunner(t *testing.T) {
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
				API:       "some/api",
				APISource: newTestGitRepo(t).GetDir(),
				Repo:      newTestGitRepo(t).GetDir(),
				WorkRoot:  t.TempDir(),
				Image:     "gcr.io/test/test-image",
			},
		},
		{
			name: "invalid config",
			cfg: &config.Config{
				APISource: newTestGitRepo(t).GetDir(),
			},
			wantErr:    true,
			wantErrMsg: "failed to create init runner",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := newInitRunner(test.cfg)
			if test.wantErr {
				if err == nil {
					t.Error("newInitRunner() should return error")
				}

				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("want error message: %q, got %q", test.wantErrMsg, err.Error())
				}

				return
			}
			if err != nil {
				t.Errorf("newInitRunner() = %v, want nil", err)
			}
		})
	}
}

func TestInitRun(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name       string
		runner     *initRunner
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "run docker command",
			runner: &initRunner{
				workRoot:        os.TempDir(),
				containerClient: &mockContainerClient{},
				cfg:             &config.Config{},
				state:           &config.LibrarianState{},
			},
		},
		{
			name: "docker command returns error",
			runner: &initRunner{
				workRoot: os.TempDir(),
				containerClient: &mockContainerClient{
					initErr: errors.New("simulated init error"),
				},
				cfg:   &config.Config{},
				state: &config.LibrarianState{},
			},
			wantErr:    true,
			wantErrMsg: "simulated init error",
		},
		{
			name: "invalid work root",
			runner: &initRunner{
				workRoot: "/invalid/path",
			},
			wantErr:    true,
			wantErrMsg: "failed to create output dir",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := test.runner.run(context.Background())
			if test.wantErr {
				if err == nil {
					t.Error("run() should return error")
				}

				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("want error message: %q, got %q", test.wantErrMsg, err.Error())
				}

				return
			}
			if err != nil {
				t.Errorf("run() got nil runner, want non-nil")
			}
		})
	}
}

func TestSetReleaseTrigger(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name                 string
		state                *config.LibrarianState
		libraryID            string
		libraryVersion       string
		trigger              bool
		wantLibraryToTrigger map[string]bool
		wantLibraryToVersion map[string]string
	}{
		{
			name: "set trigger for all libraries",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:      "one-example-id",
						Version: "1.0.0",
					},
					{
						ID:      "another-example-id",
						Version: "1.0.1",
					},
				},
			},
			trigger: true,
			wantLibraryToTrigger: map[string]bool{
				"one-example-id":     true,
				"another-example-id": true,
			},
			wantLibraryToVersion: map[string]string{
				"one-example-id":     "1.0.0",
				"another-example-id": "1.0.1",
			},
		},
		{
			name: "set trigger for one library",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:      "one-example-id",
						Version: "1.0.0",
					},
					{
						ID:      "another-example-id",
						Version: "1.0.1",
					},
				},
			},
			libraryID: "another-example-id",
			trigger:   true,
			wantLibraryToTrigger: map[string]bool{
				"one-example-id":     false,
				"another-example-id": true,
			},
			wantLibraryToVersion: map[string]string{
				"one-example-id":     "1.0.0",
				"another-example-id": "1.0.1",
			},
		},
		{
			name: "set trigger for one library and override version",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:      "one-example-id",
						Version: "1.0.0",
					},
					{
						ID:      "another-example-id",
						Version: "1.0.1",
					},
				},
			},
			libraryID:      "another-example-id",
			libraryVersion: "2.0.0",
			trigger:        true,
			wantLibraryToTrigger: map[string]bool{
				"one-example-id":     false,
				"another-example-id": true,
			},
			wantLibraryToVersion: map[string]string{
				"one-example-id":     "1.0.0",
				"another-example-id": "2.0.0",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			setReleaseTrigger(test.state, test.libraryID, test.libraryVersion, test.trigger)
			for _, library := range test.state.Libraries {
				wantTrigger, ok := test.wantLibraryToTrigger[library.ID]
				if !ok || library.ReleaseTriggered != wantTrigger {
					t.Errorf("library %s should set release trigger to %v, got %v", library.ID, test.trigger, library.ReleaseTriggered)
				}
				wantVersion, ok := test.wantLibraryToVersion[library.ID]
				if !ok || library.Version != wantVersion {
					t.Errorf("library %s should set version to %s, got %s", library.ID, test.libraryVersion, library.Version)
				}
			}
		})
	}
}
