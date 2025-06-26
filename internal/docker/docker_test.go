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

package docker

import (
	"context"
	"fmt"
	"testing"

	"github.com/googleapis/librarian/internal/config"

	"github.com/google/go-cmp/cmp"
)

func TestDockerRun(t *testing.T) {
	const (
		testAPIPath         = "testAPIPath"
		testAPIRoot         = "testAPIRoot"
		testGeneratorInput  = "testGeneratorInput"
		testGeneratorOutput = "testGeneratorOutput"
		testImage           = "testImage"
		testInputsDirectory = "testInputsDirectory"
		testLanguageRepo    = "testLanguageRepo"
		testLibraryID       = "testLibraryID"
		testLibraryVersion  = "testLibraryVersion"
		testOutput          = "testOutput"
		testOutputDir       = "testOutputDir"
		testReleaseVersion  = "testReleaseVersion"
		testRepoRoot        = "testRepoRoot"
	)

	d := &Docker{
		Image: testImage,
	}

	cfg := &config.Config{}

	for _, test := range []struct {
		name       Command
		runCommand func(ctx context.Context) error
		want       []string
	}{
		{
			name: CommandGenerateRaw,
			runCommand: func(ctx context.Context) error {
				return d.GenerateRaw(ctx, cfg, testAPIRoot, testOutput, testAPIPath)
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s:/apis", testAPIRoot),
				"-v", fmt.Sprintf("%s:/output", testOutput),
				"--network=none",
				testImage,
				string(CommandGenerateRaw),
				"--api-root=/apis",
				"--output=/output",
				fmt.Sprintf("--api-path=%s", testAPIPath),
			},
		},
		{
			name: CommandGenerateLibrary,
			runCommand: func(ctx context.Context) error {
				return d.GenerateLibrary(ctx, cfg, testAPIRoot, testOutput, testGeneratorInput, testLibraryID)
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s:/apis", testAPIRoot),
				"-v", fmt.Sprintf("%s:/output", testOutput),
				"-v", fmt.Sprintf("%s:/generator-input", testGeneratorInput),
				"--network=none",
				testImage,
				string(CommandGenerateLibrary),
				"--api-root=/apis",
				"--output=/output",
				"--generator-input=/generator-input",
				fmt.Sprintf("--library-id=%s", testLibraryID),
			},
		},
		{
			name: CommandClean,
			runCommand: func(ctx context.Context) error {
				return d.Clean(ctx, cfg, testRepoRoot, testLibraryID)
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s:/repo", testRepoRoot),
				"--network=none",
				testImage,
				string(CommandClean),
				"--repo-root=/repo",
				fmt.Sprintf("--library-id=%s", testLibraryID),
			},
		},
		{
			name: CommandBuildRaw,
			runCommand: func(ctx context.Context) error {
				return d.BuildRaw(ctx, cfg, testGeneratorOutput, testAPIPath)
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s:/generator-output", testGeneratorOutput),
				testImage,
				string(CommandBuildRaw),
				"--generator-output=/generator-output",
				fmt.Sprintf("--api-path=%s", testAPIPath),
			},
		},
		{
			name: CommandBuildLibrary,
			runCommand: func(ctx context.Context) error {
				return d.BuildLibrary(ctx, cfg, testRepoRoot, testLibraryID)
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s:/repo", testRepoRoot),
				testImage,
				string(CommandBuildLibrary),
				"--repo-root=/repo",
				"--test=true",
				fmt.Sprintf("--library-id=%s", testLibraryID),
			},
		},
		{
			name: CommandConfigure,
			runCommand: func(ctx context.Context) error {
				return d.Configure(ctx, cfg, testAPIRoot, testAPIPath, testGeneratorInput)
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s:/apis", testAPIRoot),
				"-v", fmt.Sprintf("%s:/generator-input", testGeneratorInput),
				"--network=none",
				testImage,
				string(CommandConfigure),
				"--api-root=/apis",
				"--generator-input=/generator-input",
				fmt.Sprintf("--api-path=%s", testAPIPath),
			},
		},
		{
			name: CommandPrepareLibraryRelease,
			runCommand: func(ctx context.Context) error {
				return d.PrepareLibraryRelease(ctx, cfg, testLanguageRepo, testInputsDirectory, testLibraryID, testReleaseVersion)
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s:/repo", testLanguageRepo),
				"-v", fmt.Sprintf("%s:/inputs", testInputsDirectory),
				"--network=none",
				testImage,
				string(CommandPrepareLibraryRelease),
				"--repo-root=/repo",
				fmt.Sprintf("--library-id=%s", testLibraryID),
				fmt.Sprintf("--release-notes=/inputs/%s-%s-release-notes.txt", testLibraryID, testReleaseVersion),
				fmt.Sprintf("--version=%s", testReleaseVersion),
			},
		},
		{
			name: CommandIntegrationTestLibrary,
			runCommand: func(ctx context.Context) error {
				return d.IntegrationTestLibrary(ctx, cfg, testLanguageRepo, testLibraryID)
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s:/repo", testLanguageRepo),
				testImage,
				string(CommandIntegrationTestLibrary),
				"--repo-root=/repo",
				fmt.Sprintf("--library-id=%s", testLibraryID),
			},
		},
		{
			name: CommandPackageLibrary,
			runCommand: func(ctx context.Context) error {
				return d.PackageLibrary(ctx, cfg, testLanguageRepo, testLibraryID, testOutputDir)
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s:/repo", testLanguageRepo),
				"-v", fmt.Sprintf("%s:/output", testOutputDir),
				testImage,
				string(CommandPackageLibrary),
				"--repo-root=/repo",
				"--output=/output",
				fmt.Sprintf("--library-id=%s", testLibraryID),
			},
		},
		{
			name: CommandPublishLibrary,
			runCommand: func(ctx context.Context) error {
				return d.PublishLibrary(ctx, cfg, testOutputDir, testLibraryID, testLibraryVersion)
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s:/output", testOutputDir),
				testImage,
				string(CommandPublishLibrary),
				"--package-output=/output",
				fmt.Sprintf("--library-id=%s", testLibraryID),
				fmt.Sprintf("--version=%s", testLibraryVersion),
			},
		},
	} {
		t.Run(string(test.name), func(t *testing.T) {
			d.run = func(args ...string) error {
				if diff := cmp.Diff(test.want, args); diff != "" {
					t.Errorf("mismatch(-want +got):\n%s", diff)
				}
				return nil
			}
			ctx := context.Background()
			if err := test.runCommand(ctx); err != nil {
				t.Fatal(err)
			}
		})
	}
}
