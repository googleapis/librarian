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
	"errors"
	"flag"
	"os"
	"testing"
)

func TestValidatePush(t *testing.T) {
	tests := []struct {
		name           string
		setFlagPush    bool
		setGitHubToken bool
		expectedErr    error
	}{
		{
			name:           "push is false, no error",
			setFlagPush:    false,
			setGitHubToken: false,
			expectedErr:    nil,
		},
		{
			name:           "push is true, token exists, no error",
			setFlagPush:    true,
			setGitHubToken: true,
			expectedErr:    nil,
		},
		{
			name:           "push is true, no token, error",
			setFlagPush:    true,
			setGitHubToken: false,
			expectedErr:    errors.New("no GitHub token supplied for push"),
		},
		{
			name:           "push is false, no token, no error",
			setFlagPush:    false,
			setGitHubToken: true,
			expectedErr:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetAllFlags()

			// Set up global flag and environment variable based on test case
			flagPush = tt.setFlagPush
			var cleanupEnv func()
			if tt.setGitHubToken {
				cleanupEnv = setEnvVar(t, "LIBRARIAN_GITHUB_TOKEN", "mock_token_abc")
			} else {
				cleanupEnv = setEnvVar(t, "LIBRARIAN_GITHUB_TOKEN", "")
			}
			defer cleanupEnv()

			err := validatePush()

			if (err != nil) != (tt.expectedErr != nil) {
				t.Errorf("validatePush() error = %v, wantErr %v", err, tt.expectedErr)
			} else if tt.expectedErr != nil && err.Error() != tt.expectedErr.Error() {
				t.Errorf("validatePush() error message = %q, want %q", err.Error(), tt.expectedErr.Error())
			}
		})
	}
}

func TestValidateSkipIntegrationTests(t *testing.T) {
	tests := []struct {
		name                        string
		setFlagSkipIntegrationTests string
		expectedErr                 error
	}{
		{
			name:                        "empty string, no error",
			setFlagSkipIntegrationTests: "",
			expectedErr:                 nil,
		},
		{
			name:                        "valid bug string, no error",
			setFlagSkipIntegrationTests: "b/12345",
			expectedErr:                 nil,
		},
		{
			name:                        "invalid string (no prefix), error",
			setFlagSkipIntegrationTests: "12345",
			expectedErr:                 errors.New("skipping integration tests requires a bug to be specified, e.g. -skip-integration-tests=b/12345"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetAllFlags()
			flagSkipIntegrationTests = tt.setFlagSkipIntegrationTests

			err := validateSkipIntegrationTests()

			if (err != nil) != (tt.expectedErr != nil) {
				t.Errorf("validateSkipIntegrationTests() error = %v, wantErr %v", err, tt.expectedErr)
			} else if tt.expectedErr != nil && err.Error() != tt.expectedErr.Error() {
				t.Errorf("validateSkipIntegrationTests() error message = %q, want %q", err.Error(), tt.expectedErr.Error())
			}
		})
	}
}

func TestValidateRequiredFlag(t *testing.T) {
	tests := []struct {
		name        string
		flagName    string
		flagValue   string
		expectedErr error
	}{
		{
			name:        "flag provided, no error",
			flagName:    "api-path",
			flagValue:   "google/cloud/speech/v1",
			expectedErr: nil,
		},
		{
			name:        "flag not provided, error",
			flagName:    "api-root",
			flagValue:   "",
			expectedErr: errors.New("required flag -api-root not specified"),
		},
		{
			name:        "flag with spaces, error",
			flagName:    "language",
			flagValue:   " ",
			expectedErr: errors.New("required flag -language not specified"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This function doesn't rely on global flags directly,
			// but rather on parameters, so no need for resetAllFlags here specifically.
			err := validateRequiredFlag(tt.flagName, tt.flagValue)

			if (err != nil) != (tt.expectedErr != nil) {
				t.Errorf("validateRequiredFlag() error = %v, wantErr %v", err, tt.expectedErr)
			} else if tt.expectedErr != nil && err.Error() != tt.expectedErr.Error() {
				t.Errorf("validateRequiredFlag() error message = %q, want %q", err.Error(), tt.expectedErr.Error())
			}
		})
	}
}

// Helper function to reset global flags after each test.
func resetAllFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	flagAPIPath = ""
	flagAPIRoot = ""
	flagArtifactRoot = ""
	flagBaselineCommit = ""
	flagBranch = ""
	flagBuild = false
	flagEnvFile = ""
	flagGitUserEmail = ""
	flagGitUserName = ""
	flagImage = ""
	flagLanguage = ""
	flagLibraryID = ""
	flagLibraryVersion = ""
	flagPush = false
	flagReleaseID = ""
	flagReleasePRUrl = ""
	flagRepoRoot = ""
	flagRepoUrl = ""
	flagSecretsProject = ""
	flagSkipIntegrationTests = ""
	flagSyncUrlPrefix = ""
	flagTag = ""
	flagTagRepoUrl = ""
	flagWorkRoot = ""
}

// Mocking helper for environment variables
// Returns a cleanup function.
func setEnvVar(t *testing.T, key, value string) func() {
	originalValue, exists := os.LookupEnv(key)
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("Failed to set environment variable %s: %v", key, err)
	}

	return func() {
		if exists {
			os.Setenv(key, originalValue)
		} else {
			os.Unsetenv(key)
		}
	}
}
