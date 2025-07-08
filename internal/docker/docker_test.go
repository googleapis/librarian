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

func TestNew(t *testing.T) {
	const (
		testWorkRoot       = "testWorkRoot"
		testImage          = "testImage"
		testSecretsProject = "testSecretsProject"
		testUID            = "1000"
		testGID            = "1001"
	)
	pipelineConfig := &config.PipelineConfig{}
	d, err := New(testWorkRoot, testImage, testSecretsProject, testUID, testGID, pipelineConfig)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if d.Image != testImage {
		t.Errorf("d.Image = %q, want %q", d.Image, testImage)
	}
	if d.uid != testUID {
		t.Errorf("d.uid = %q, want %q", d.uid, testUID)
	}
	if d.gid != testGID {
		t.Errorf("d.gid = %q, want %q", d.gid, testGID)
	}
	if d.env == nil {
		t.Error("d.env is nil")
	}
	if d.run == nil {
		t.Error("d.run is nil")
	}
}

func TestDockerRun(t *testing.T) {
	const (
		testUID             = "1000"
		testGID             = "1001"
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
		testReleaseVersion  = "testReleaseVersion"
		testRepoRoot        = "testRepoRoot"
	)

	cfg := &config.Config{}
	cfgInDocker := &config.Config{
		HostMount: "hostDir:localDir",
	}
	for _, test := range []struct {
		name       string
		docker     *Docker
		runCommand func(ctx context.Context, d *Docker) error
		want       []string
	}{
		{
			name: "GenerateRaw",
			docker: &Docker{
				Image: testImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
				return d.GenerateRaw(ctx, cfg, testAPIRoot, testOutput, testAPIPath)
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s:/apis", testAPIRoot),
				"-v", fmt.Sprintf("%s:/output", testOutput),
				"--network=none",
				testImage,
				string(CommandGenerateRaw),
				"--source=/apis",
				"--output=/output",
				fmt.Sprintf("--api=%s", testAPIPath),
			},
		},
		{
			name: "GenerateRaw with user",
			docker: &Docker{
				Image: testImage,
				uid:   testUID,
				gid:   testGID,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
				return d.GenerateRaw(ctx, cfg, testAPIRoot, testOutput, testAPIPath)
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s:/apis", testAPIRoot),
				"-v", fmt.Sprintf("%s:/output", testOutput),
				"--user", fmt.Sprintf("%s:%s", testUID, testGID),
				"--network=none",
				testImage,
				string(CommandGenerateRaw),
				"--source=/apis",
				"--output=/output",
				fmt.Sprintf("--api=%s", testAPIPath),
			},
		},
		{
			name: "GenerateLibrary",
			docker: &Docker{
				Image: testImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
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
				"--source=/apis",
				"--output=/output",
				"--generator-input=/generator-input",
				fmt.Sprintf("--library-id=%s", testLibraryID),
			},
		},
		{
			name: "GenerateLibrary runs in docker",
			docker: &Docker{
				Image: testImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
				return d.GenerateLibrary(ctx, cfgInDocker, testAPIRoot, "hostDir", testGeneratorInput, testLibraryID)
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s:/apis", testAPIRoot),
				"-v", "localDir:/output",
				"-v", fmt.Sprintf("%s:/generator-input", testGeneratorInput),
				"--network=none",
				testImage,
				string(CommandGenerateLibrary),
				"--source=/apis",
				"--output=/output",
				"--generator-input=/generator-input",
				fmt.Sprintf("--library-id=%s", testLibraryID),
			},
		},
		{
			name: "Clean",
			docker: &Docker{
				Image: testImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
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
			name: "BuildRaw",
			docker: &Docker{
				Image: testImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
				return d.BuildRaw(ctx, cfg, testGeneratorOutput, testAPIPath)
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s:/generator-output", testGeneratorOutput),
				testImage,
				string(CommandBuildRaw),
				"--generator-output=/generator-output",
				fmt.Sprintf("--api=%s", testAPIPath),
			},
		},
		{
			name: "BuildLibrary",
			docker: &Docker{
				Image: testImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
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
			name: "Configure",
			docker: &Docker{
				Image: testImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
				return d.Configure(ctx, cfg, testAPIRoot, testAPIPath, testGeneratorInput)
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s:/apis", testAPIRoot),
				"-v", fmt.Sprintf("%s:/generator-input", testGeneratorInput),
				"--network=none",
				testImage,
				string(CommandConfigure),
				"--source=/apis",
				"--generator-input=/generator-input",
				fmt.Sprintf("--api=%s", testAPIPath),
			},
		},
		{
			name: "PrepareLibraryRelease",
			docker: &Docker{
				Image: testImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
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
			name: "IntegrationTestLibrary",
			docker: &Docker{
				Image: testImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
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
			name: "PackageLibrary",
			docker: &Docker{
				Image: testImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
				return d.PackageLibrary(ctx, cfg, testLanguageRepo, testLibraryID, testOutput)
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s:/repo", testLanguageRepo),
				"-v", fmt.Sprintf("%s:/output", testOutput),
				testImage,
				string(CommandPackageLibrary),
				"--repo-root=/repo",
				"--output=/output",
				fmt.Sprintf("--library-id=%s", testLibraryID),
			},
		},
		{
			name: "PublishLibrary",
			docker: &Docker{
				Image: testImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
				return d.PublishLibrary(ctx, cfg, testOutput, testLibraryID, testLibraryVersion)
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s:/output", testOutput),
				testImage,
				string(CommandPublishLibrary),
				"--package-output=/output",
				fmt.Sprintf("--library-id=%s", testLibraryID),
				fmt.Sprintf("--version=%s", testLibraryVersion),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.docker.run = func(args ...string) error {
				if diff := cmp.Diff(test.want, args); diff != "" {
					t.Errorf("mismatch(-want +got):\n%s", diff)
				}
				return nil
			}
			ctx := t.Context()
			if err := test.runCommand(ctx, test.docker); err != nil {
				t.Fatal(err)
			}
		})
	}
}
