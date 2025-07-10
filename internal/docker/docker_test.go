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
			name: "Generate",
			docker: &Docker{
				Image: testImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
				return d.Generate(ctx, cfg, testAPIRoot, testOutput, testGeneratorInput, testLibraryID)
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s:/apis", testAPIRoot),
				"-v", fmt.Sprintf("%s:/output", testOutput),
				"-v", fmt.Sprintf("%s:/generator-input", testGeneratorInput),
				testImage,
				string(CommandGenerate),
				"--source=/apis",
				"--output=/output",
				"--generator-input=/generator-input",
				fmt.Sprintf("--library-id=%s", testLibraryID),
			},
		},
		{
			name: "Generate runs in docker",
			docker: &Docker{
				Image: testImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
				return d.Generate(ctx, cfgInDocker, testAPIRoot, "hostDir", testGeneratorInput, testLibraryID)
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s:/apis", testAPIRoot),
				"-v", "localDir:/output",
				"-v", fmt.Sprintf("%s:/generator-input", testGeneratorInput),
				testImage,
				string(CommandGenerate),
				"--source=/apis",
				"--output=/output",
				"--generator-input=/generator-input",
				fmt.Sprintf("--library-id=%s", testLibraryID),
			},
		},
		{
			name: "Build",
			docker: &Docker{
				Image: testImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
				return d.Build(ctx, cfg, testRepoRoot, testLibraryID)
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s:/repo", testRepoRoot),
				testImage,
				string(CommandBuild),
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
				testImage,
				string(CommandConfigure),
				"--source=/apis",
				"--generator-input=/generator-input",
				fmt.Sprintf("--api=%s", testAPIPath),
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
