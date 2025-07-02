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

package config

import (
	"errors"
	"os/user"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNew(t *testing.T) {
	for _, test := range []struct {
		name    string
		envVars map[string]string
		want    Config
	}{
		{
			name: "All environment variables set",
			envVars: map[string]string{
				"LIBRARIAN_HOST_ROOT_DIR":   "/host/root",
				"LIBRARIAN_ROOT_DIR":        "/mount/root",
				"LIBRARIAN_GITHUB_TOKEN":    "gh_token",
				"LIBRARIAN_SYNC_AUTH_TOKEN": "sync_token",
			},
			want: Config{
				DockerHostRootDir:  "/host/root",
				DockerMountRootDir: "/mount/root",
				GitHubToken:        "gh_token",
				SyncAuthToken:      "sync_token",
			},
		},
		{
			name:    "No environment variables set",
			envVars: map[string]string{},
			want: Config{
				DockerHostRootDir:  "",
				DockerMountRootDir: "",
				GitHubToken:        "",
				SyncAuthToken:      "",
			},
		},
		{
			name: "Some environment variables set",
			envVars: map[string]string{
				"LIBRARIAN_HOST_ROOT_DIR": "/host/root",
				"LIBRARIAN_GITHUB_TOKEN":  "gh_token",
			},
			want: Config{
				DockerHostRootDir:  "/host/root",
				DockerMountRootDir: "",
				GitHubToken:        "gh_token",
				SyncAuthToken:      "",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			for k, v := range test.envVars {
				t.Setenv(k, v)
			}

			got := New()

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
			err := cfg.SetupUser()

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
		name      string
		cfg       Config
		wantValid bool
		wantErr   bool
	}{
		{
			name: "Valid config - Push false, push config valid",
			cfg: Config{
				Push:        false,
				GitHubToken: "",
				PushConfig:  "def@ghi.com,abc",
			},
			wantValid: true,
			wantErr:   false,
		},
		{
			name: "Valid config - Push true, token present, push config valid",
			cfg: Config{
				Push:        true,
				GitHubToken: "some_token",
				PushConfig:  "def@ghi.com,abc",
			},
			wantValid: true,
			wantErr:   false,
		},
		{
			name: "Invalid config - Push true, token missing, push config valid",
			cfg: Config{
				Push:        true,
				GitHubToken: "",
				PushConfig:  "def@ghi.com,abc",
			},
			wantValid: false,
			wantErr:   true,
		},
		{
			name: "Invalid config - Push true, token present, push config invalid",
			cfg: Config{
				Push:        true,
				GitHubToken: "some_token",
				PushConfig:  "abc:def@ghi.com",
			},
			wantValid: false,
			wantErr:   true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotValid, err := test.cfg.IsValid()

			if gotValid != test.wantValid {
				t.Errorf("IsValid() got valid = %t, want %t", gotValid, test.wantValid)
			}

			if (err != nil) != test.wantErr {
				t.Errorf("IsValid() got error = %v, want error = %t", err, test.wantErr)
			}
			if test.wantErr && err != nil && err.Error() != "no GitHub token supplied for push" && err.Error() != "unable to parse push config" {
				t.Errorf("IsValid() got unexpected error message: %q", err.Error())
			}
		})
	}
}
