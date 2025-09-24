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
// WITHOUT WARRANTIES, OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNew(t *testing.T) {
	t.Setenv("LIBRARIAN_GITHUB_TOKEN", "")
	for _, test := range []struct {
		name    string
		envVars map[string]string
		want    Config
	}{
		{
			name: "All environment variables set",
			envVars: map[string]string{
				"LIBRARIAN_GITHUB_TOKEN":    "gh_token",
				"LIBRARIAN_SYNC_AUTH_TOKEN": "sync_token",
			},
			want: Config{
				GitHubToken: "gh_token",
				CommandName: "test",
			},
		},
		{
			name:    "No environment variables set",
			envVars: map[string]string{},
			want: Config{
				GitHubToken: "",
				CommandName: "test",
			},
		},
		{
			name: "Some environment variables set",
			envVars: map[string]string{
				"LIBRARIAN_GITHUB_TOKEN": "gh_token",
			},
			want: Config{
				GitHubToken: "gh_token",
				CommandName: "test",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			for k, v := range test.envVars {
				t.Setenv(k, v)
			}

			got := New("test")

			if diff := cmp.Diff(&test.want, got); diff != "" {
				t.Errorf("New() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSetupUser(t *testing.T) {
	originalCurrentUser := currentUser
	t.Cleanup(func() {
		currentUser = originalCurrentUser
	})

	for _, test := range []struct {
		name     string
		mockUser *user.User
		mockErr  error
		wantUID  string
		wantGID  string
		wantErr  bool
	}{
		{
			name: "Success",
			mockUser: &user.User{
				Uid: "1001",
				Gid: "1002",
			},
			mockErr: nil,
			wantUID: "1001",
			wantGID: "1002",
			wantErr: false,
		},
		{
			name:     "Error getting user",
			mockUser: nil,
			mockErr:  errors.New("user lookup failed"),
			wantUID:  "",
			wantGID:  "",
			wantErr:  true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			currentUser = func() (*user.User, error) {
				return test.mockUser, test.mockErr
			}

			cfg := &Config{}
			err := cfg.setupUser()

			if (err != nil) != test.wantErr {
				t.Errorf("SetupUser() error = %v, wantErr %v", err, test.wantErr)
				return
			}

			if cfg.UserUID != test.wantUID {
				t.Errorf("SetupUser() got UID = %q, want %q", cfg.UserUID, test.wantUID)
			}
			if cfg.UserGID != test.wantGID {
				t.Errorf("SetupUser() got GID = %q, want %q", cfg.UserGID, test.wantGID)
			}
		})
	}
}

func TestIsValid(t *testing.T) {
	for _, test := range []struct {
		name       string
		cfg        Config
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "Valid config - Push false",
			cfg: Config{
				Push: false,
				Repo: "/tmp/some/repo",
			},
		},
		{
			name: "Valid config - Push true, token present",
			cfg: Config{
				Push:        true,
				GitHubToken: "some_token",
				Repo:        "/tmp/some/repo",
			},
		},
		{
			name: "Valid config - missing library version",
			cfg: Config{
				Push:        true,
				GitHubToken: "some_token",
				Library:     "library-id",
				Repo:        "/tmp/some/repo",
			},
		},
		{
			name: "Valid config - valid pull request",
			cfg: Config{
				PullRequest: "https://github.com/owner/repo/pull/123",
				Repo:        "/tmp/some/repo",
			},
		},
		{
			name: "Invalid config - Push true, token missing",
			cfg: Config{
				Push:        true,
				GitHubToken: "",
				Repo:        "/tmp/some/repo",
			},
			wantErr:    true,
			wantErrMsg: "no GitHub token supplied for push",
		},
		{
			name: "Invalid config - library version presents, missing library id",
			cfg: Config{
				Push:           true,
				GitHubToken:    "some_token",
				LibraryVersion: "1.2.3",
				Repo:           "/tmp/some/repo",
			},
			wantErr:    true,
			wantErrMsg: "specified library version without library id",
		},
		{
			name: "Invalid config - host mount invalid, missing local-dir",
			cfg: Config{
				HostMount: "host-dir:",
				Repo:      "/tmp/some/repo",
			},
			wantErr:    true,
			wantErrMsg: "unable to parse host mount",
		},
		{
			name: "Invalid config - host mount invalid, missing host-dir",
			cfg: Config{
				HostMount: ":local-dir",
				Repo:      "/tmp/some/repo",
			},
			wantErr:    true,
			wantErrMsg: "unable to parse host mount",
		},
		{
			name: "Invalid config - host mount invalid, missing separator",
			cfg: Config{
				HostMount: "host-dir/local-dir",
				Repo:      "/tmp/some/repo",
			},
			wantErr:    true,
			wantErrMsg: "unable to parse host mount",
		},
		{
			name: "Invalid config -  missing Repo",
			cfg: Config{
				Repo: "",
			},
			wantErr:    true,
			wantErrMsg: "language repository not specified or detected",
		},
		{
			name: "Invalid config - invalid pull request url",
			cfg: Config{
				PullRequest: "https://github.com/owner/repo/issues/123",
			},
			wantErr:    true,
			wantErrMsg: "pull request URL is not valid",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotValid, err := test.cfg.IsValid()

			if gotValid != !test.wantErr {
				t.Errorf("IsValid() got valid = %t, want %t", gotValid, !test.wantErr)
			}

			if test.wantErr && !strings.Contains(err.Error(), test.wantErrMsg) {
				t.Errorf("IsValid() got unexpected error message: %q", err.Error())
			}
		})
	}
}

func TestCreateWorkRoot(t *testing.T) {
	localTempDir := t.TempDir()
	tempDir = func() string {
		return localTempDir
	}
	t.Cleanup(func() {
		tempDir = os.TempDir
	})
	for _, test := range []struct {
		name   string
		config *Config
		setup  func(t *testing.T) (string, func())
	}{
		{
			name: "configured root",
			config: &Config{
				WorkRoot: "/some/path",
			},
			setup: func(t *testing.T) (string, func()) {
				return "/some/path", func() {}
			},
		},
		{
			name: "version command",
			config: &Config{
				CommandName: "version",
				WorkRoot:    "/some/path",
			},
			setup: func(t *testing.T) (string, func()) {
				return "/some/path", func() {}
			},
		},
		{
			name:   "without override, new dir",
			config: &Config{},
			setup: func(t *testing.T) (string, func()) {
				expectedPath := filepath.Join(localTempDir, "librarian-")
				return expectedPath, func() {
					if err := os.RemoveAll(expectedPath); err != nil {
						t.Errorf("os.RemoveAll(%q) = %v; want nil", expectedPath, err)
					}
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			want, cleanup := test.setup(t)
			defer cleanup()

			if err := test.config.createWorkRoot(); err != nil {
				t.Errorf("createWorkRoot() got unexpected error: %v", err)
				return
			}

			if !strings.HasPrefix(test.config.WorkRoot, want) {
				t.Errorf("createWorkRoot() = %v, want %v", test.config.WorkRoot, want)
			}
		})
	}
}

func TestCreateWorkRootError(t *testing.T) {
	tempDir = func() string {
		return filepath.Join("--invalid--", "--not-a-directory--")
	}
	t.Cleanup(func() {
		tempDir = os.TempDir
	})
	config := &Config{}
	if err := config.createWorkRoot(); err == nil {
		t.Errorf("createWorkRoot() expected an error got: %v", config.WorkRoot)
	}
}

func TestDeriveRepo(t *testing.T) {
	for _, test := range []struct {
		name         string
		config       *Config
		setup        func(t *testing.T, dir string)
		wantErr      bool
		wantRepoPath string
	}{
		{
			name: "configured repo path",
			config: &Config{
				Repo: "/some/path",
			},
			wantRepoPath: "/some/path",
		},
		{
			name:   "empty repo path, state file exists",
			config: &Config{},
			setup: func(t *testing.T, dir string) {
				stateDir := filepath.Join(dir, LibrarianDir)
				if err := os.MkdirAll(stateDir, 0755); err != nil {
					t.Fatal(err)
				}
				stateFile := filepath.Join(stateDir, LibrarianStateFile)
				if err := os.WriteFile(stateFile, []byte("test"), 0644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name:    "empty repo path, no state file",
			config:  &Config{},
			wantErr: true,
		},
		{
			name: "version command",
			config: &Config{
				Repo:        "/some/path",
				CommandName: "version",
			},
			wantRepoPath: "/some/path",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			if test.setup != nil {
				test.setup(t, tmpDir)
			}
			t.Chdir(tmpDir)

			err := test.config.deriveRepo()
			if (err != nil) != test.wantErr {
				t.Errorf("deriveRepoPath() error = %v, wantErr %v", err, test.wantErr)
				return
			}

			wantPath := test.wantRepoPath
			if wantPath == "" && !test.wantErr {
				wantPath = tmpDir
			}

			if diff := cmp.Diff(wantPath, test.config.Repo); diff != "" {
				t.Errorf("deriveRepoPath() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSetDefaults(t *testing.T) {
	currentUser = func() (*user.User, error) {
		return &user.User{
			Uid: "1001",
			Gid: "1002",
		}, nil
	}

	t.Cleanup(func() {
		currentUser = user.Current
	})
	for _, test := range []struct {
		name     string
		setup    func(t *testing.T, dir string)
		workRoot string
		repoRoot string
		wantErr  bool
	}{
		{
			name: "all defaults",
			setup: func(t *testing.T, dir string) {
				stateDir := filepath.Join(dir, LibrarianDir)
				if err := os.MkdirAll(stateDir, 0755); err != nil {
					t.Fatal(err)
				}
				stateFile := filepath.Join(stateDir, LibrarianStateFile)
				if err := os.WriteFile(stateFile, []byte("test"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			workRoot: "",
			repoRoot: "",
		},
		{
			name: "work root specified",
			setup: func(t *testing.T, dir string) {
				stateDir := filepath.Join(dir, LibrarianDir)
				if err := os.MkdirAll(stateDir, 0755); err != nil {
					t.Fatal(err)
				}
				stateFile := filepath.Join(stateDir, LibrarianStateFile)
				if err := os.WriteFile(stateFile, []byte("test"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			workRoot: "/tmp/some/path",
			repoRoot: "",
		},
		{
			name:     "repo root specified",
			workRoot: "",
			repoRoot: "/tmp/my-repo-root",
		},
		{
			name:     "all specified",
			workRoot: "/tmp/my-work-root",
			repoRoot: "/tmp/my-repo-root",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			localTempDir := t.TempDir()
			tempDir = func() string {
				return localTempDir
			}
			t.Cleanup(func() {
				tempDir = os.TempDir
			})
			if test.setup != nil {
				test.setup(t, localTempDir)
				t.Chdir(localTempDir)
			}
			cfg := &Config{
				WorkRoot: test.workRoot,
				Repo:     test.repoRoot,
			}

			err := cfg.SetDefaults()
			if (err != nil) != test.wantErr {
				t.Errorf("SetDefaults() error = %v, wantErr %v", err, test.wantErr)
				return
			}

			if test.wantErr {
				return
			}

			if cfg.UserUID == "" || cfg.UserGID == "" {
				t.Errorf("User UID/GID not set")
			}

			if test.workRoot == "" && cfg.WorkRoot == "" {
				t.Errorf("WorkRoot not set")
			}

			if test.repoRoot == "" && cfg.Repo == "" {
				t.Errorf("Repo not set")
			}
		})
	}
}

func TestValidateHostMount(t *testing.T) {
	for _, test := range []struct {
		name         string
		hostMount    string
		defaultMount string
		wantErr      bool
		wantErrMsg   string
	}{
		{
			name:         "default host mount",
			hostMount:    "example/path:/path",
			defaultMount: "example/path:/path",
		},
		{
			name:         "valid host mount",
			hostMount:    "example/path:/mounted/path",
			defaultMount: "another/path:/path",
		},
		{
			name:         "invalid host mount",
			hostMount:    "example/path",
			defaultMount: "example/path:/path",
			wantErr:      true,
			wantErrMsg:   "unable to parse host mount",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ok, err := validateHostMount(test.hostMount, test.defaultMount)
			if test.wantErr {
				if err == nil {
					t.Fatal("validateHostMount() should return error")
				}

				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("want error message: %q, got %q", test.wantErrMsg, err.Error())
				}

				return
			}

			if !ok || err != nil {
				t.Error("validateHostMount() should not return error")
			}
		})
	}
}
