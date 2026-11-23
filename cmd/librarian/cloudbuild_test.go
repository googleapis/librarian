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

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/yaml"
)

// dockerBuilder is the digest-pinned builder every file must use.
const dockerBuilder = "gcr.io/cloud-builders/docker@sha256:56fbada6132e04ae6d9bcd2d7e2129d945ecfbd909e3ca4c09141f5fec25de16"

// imageLanguages lists the languages whose librarian image Cloud Build
// publishes, one cloudbuild/<language>.yaml each, because the Louhi flow passes
// no substitutions.
var imageLanguages = []string{"go", "java", "nodejs", "php", "python", "ruby", "rust", "swift"}

func TestCloudBuildFiles(t *testing.T) {
	entries, err := os.ReadDir("cloudbuild")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, entry := range entries {
		got = append(got, strings.TrimSuffix(entry.Name(), ".yaml"))
	}
	if diff := cmp.Diff(imageLanguages, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestCloudBuildFile(t *testing.T) {
	for _, language := range imageLanguages {
		t.Run(language, func(t *testing.T) {
			got, err := yaml.Read[cloudBuildConfig](filepath.Join("cloudbuild", language+".yaml"))
			if err != nil {
				t.Fatal(err)
			}
			// The AR Exit Gate's entry repo.
			image := "us-central1-docker.pkg.dev/cloud-sdk-production-pipeline/images-dev/librarian-env-" + language
			want := &cloudBuildConfig{
				Steps: []cloudBuildStep{{
					Name: dockerBuilder,
					Args: []string{"build", "--target", language, "-f", "cmd/librarian/Dockerfile", "-t", image, "."},
				}},
				Images:  []string{image},
				Timeout: "7200s",
				Options: cloudBuildOptions{
					RequestedVerifyOption: "VERIFIED",
					MachineType:           "E2_HIGHCPU_8",
				},
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

type cloudBuildConfig struct {
	Steps   []cloudBuildStep  `yaml:"steps"`
	Images  []string          `yaml:"images"`
	Timeout string            `yaml:"timeout"`
	Options cloudBuildOptions `yaml:"options"`
}

type cloudBuildStep struct {
	Name string   `yaml:"name"`
	Env  []string `yaml:"env"`
	Args []string `yaml:"args"`
}

type cloudBuildOptions struct {
	RequestedVerifyOption string `yaml:"requestedVerifyOption"`
	MachineType           string `yaml:"machineType"`
}
