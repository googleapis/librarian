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
	"os/user"
	"strings"
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
				"LIBRARIAN_GITHUB_TOKEN":    "gh_token",
				"LIBRARIAN_SYNC_AUTH_TOKEN": "sync_token",
			},
			want: Config{
				GitHubToken: "gh_token",
			},
		},
		{
			name:    "No environment variables set",
			envVars: map[string]string{},
			want: Config{
				GitHubToken: "",
			},
		},
		{
			name: "Some environment variables set",
			envVars: map[string]string{
				"LIBRARIAN_GITHUB_TOKEN": "gh_token",
			},
			want: Config{
				GitHubToken: "gh_token",
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
		name       string
		cfg        Config
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "Valid config - Push false",
			cfg: Config{
				Push:        false,
				GitHubToken: "",
			},
		},
		{
			name: "Valid config - Push true, token present",
			cfg: Config{
				Push:        true,
				GitHubToken: "some_token",
			},
		},
		{
			name: "Valid config - missing library version",
			cfg: Config{
				Push:        true,
				GitHubToken: "some_token",
				Library:     "library-id",
			},
		},
		{
			name: "Valid config - valid pull request",
			cfg: Config{
				PullRequest: "https://github.com/owner/repo/pull/123",
			},
		},
		{
			name: "Invalid config - Push true, token missing",
			cfg: Config{
				Push:        true,
				GitHubToken: "",
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
			},
			wantErr:    true,
			wantErrMsg: "specified library version without library id",
		},
		{
			name: "Invalid config - host mount invalid, missing local-dir",
			cfg: Config{
				Push:      false,
				HostMount: "host-dir:",
			},
			wantErr:    true,
			wantErrMsg: "unable to parse host mount",
		},
		{
			name: "Invalid config - host mount invalid, missing host-dir",
			cfg: Config{
				Push:      false,
				HostMount: ":local-dir",
			},
			wantErr:    true,
			wantErrMsg: "unable to parse host mount",
		},
		{
			name: "Invalid config - host mount invalid, missing separator",
			cfg: Config{
				Push:      false,
				HostMount: "host-dir/local-dir",
			},
			wantErr:    true,
			wantErrMsg: "unable to parse host mount",
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
