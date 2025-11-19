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

package automation

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRepositoriesConfig_Validate(t *testing.T) {
	for _, test := range []struct {
		name    string
		config  *RepositoriesConfig
		wantErr bool
	}{
		{
			name: "valid state",
			config: &RepositoriesConfig{
				ImageSHA: "example-sha",
				Repositories: []*RepositoryConfig{
					{
						Name:              "google-cloud-foo",
						SecretName:        "google-cloud-foo-github-token",
						SupportedCommands: []string{"generate", "stage-release", "publish-release"},
					},
				},
			},
		},
		{
			name: "valid full name",
			config: &RepositoriesConfig{
				ImageSHA: "example-sha",
				Repositories: []*RepositoryConfig{
					{
						FullName:          "https://github.com/googleapis/google-cloud-foo",
						SecretName:        "google-cloud-foo-github-token",
						SupportedCommands: []string{"generate", "stage-release", "publish-release"},
					},
				},
			},
		},
		{
			name: "missing name",
			config: &RepositoriesConfig{
				ImageSHA: "example-sha",
				Repositories: []*RepositoryConfig{
					{
						SecretName:        "google-cloud-foo-github-token",
						SupportedCommands: []string{"generate", "stage-release", "publish-release"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing secret name",
			config: &RepositoriesConfig{
				ImageSHA: "example-sha",
				Repositories: []*RepositoryConfig{
					{
						Name:              "google-cloud-foo",
						SupportedCommands: []string{"generate", "stage-release", "publish-release"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing commands",
			config: &RepositoriesConfig{
				ImageSHA: "example-sha",
				Repositories: []*RepositoryConfig{
					{
						Name:       "google-cloud-foo",
						SecretName: "google-cloud-foo-github-token",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "empty commands",
			config: &RepositoriesConfig{
				ImageSHA: "example-sha",
				Repositories: []*RepositoryConfig{
					{
						Name:              "google-cloud-foo",
						SecretName:        "google-cloud-foo-github-token",
						SupportedCommands: []string{},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid command",
			config: &RepositoriesConfig{
				ImageSHA: "example-sha",
				Repositories: []*RepositoryConfig{
					{
						Name:              "google-cloud-foo",
						SecretName:        "google-cloud-foo-github-token",
						SupportedCommands: []string{"generate", "invalid", "publish-release"},
					},
				},
			},
			wantErr: true,
		},
		{
			name:    "empty image sha",
			config:  &RepositoriesConfig{},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.config.Validate(); (err != nil) != test.wantErr {
				t.Errorf("LibrarianState.Validate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestParseRepositoriesConfig(t *testing.T) {
	for _, test := range []struct {
		name    string
		content string
		want    *RepositoriesConfig
		wantErr bool
	}{
		{
			name: "valid state",
			content: `repositories:
  - name: google-cloud-python
    github-token-secret-name: google-cloud-python-github-token
    supported-commands:
      - generate
      - stage-release
`,
			want: &RepositoriesConfig{
				Repositories: []*RepositoryConfig{
					{
						Name:              "google-cloud-python",
						SecretName:        "google-cloud-python-github-token",
						SupportedCommands: []string{"generate", "stage-release"},
					},
				},
			},
		},
		{
			name: "valid state with full name",
			content: `repositories:
  - name: google-cloud-python
    full-name: https://github.com/some-org/google-cloud-python
    github-token-secret-name: google-cloud-python-github-token
    supported-commands:
      - generate
      - stage-release
`,
			want: &RepositoriesConfig{
				Repositories: []*RepositoryConfig{
					{
						Name:              "google-cloud-python",
						FullName:          "https://github.com/some-org/google-cloud-python",
						SecretName:        "google-cloud-python-github-token",
						SupportedCommands: []string{"generate", "stage-release"},
					},
				},
			},
		},
		{
			name: "valid state with branch",
			content: `repositories:
  - name: google-cloud-python
    branch: preview
    github-token-secret-name: google-cloud-python-github-token
    supported-commands:
      - generate
      - stage-release
`,
			want: &RepositoriesConfig{
				Repositories: []*RepositoryConfig{
					{
						Name:              "google-cloud-python",
						Branch:            "preview",
						SecretName:        "google-cloud-python-github-token",
						SupportedCommands: []string{"generate", "stage-release"},
					},
				},
			},
		},
		{
			name: "invalid yaml",
			content: `repositories:
  - name: google-cloud-python
      github-token-secret-name: google-cloud-python-github-token # bad indent
    supported-commands:
      - generate
      - stage-release
`,
			want:    nil,
			wantErr: true,
		},
		{
			name: "validation error",
			content: `repositories:
  - name: google-cloud-python
    github-token-secret-name: google-cloud-python-github-token
		# missing supported-commands
`,
			want:    nil,
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			contentLoader := func(path string) ([]byte, error) {
				return []byte(path), nil
			}
			got, err := parseRepositoriesConfig(contentLoader, test.content)
			if (err != nil) != test.wantErr {
				t.Errorf("parseRepositoriesConfig() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("parseRepositoriesConfig() mismatch (-want +got): %s", diff)
			}
		})
	}
}

func TestRepositoriesForCommand(t *testing.T) {
	testConfig := &RepositoriesConfig{
		Repositories: []*RepositoryConfig{
			{
				Name:              "google-cloud-python",
				SupportedCommands: []string{"generate", "release"},
			},
			{
				Name:              "google-cloud-ruby",
				SupportedCommands: []string{"release"},
			},
			{
				Name:              "google-cloud-dotnet",
				SupportedCommands: []string{"generate", "release"},
			},
		},
	}
	for _, test := range []struct {
		name    string
		command string
		want    []string
	}{
		{
			name:    "finds subset",
			command: "generate",
			want:    []string{"google-cloud-python", "google-cloud-dotnet"},
		},
		{
			name:    "finds all",
			command: "release",
			want:    []string{"google-cloud-python", "google-cloud-ruby", "google-cloud-dotnet"},
		},
		{
			name:    "finds none",
			command: "non-existent",
			want:    []string{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := testConfig.RepositoriesForCommand(test.command)
			var names = make([]string, 0)
			for _, r := range got {
				names = append(names, r.Name)
			}
			if diff := cmp.Diff(test.want, names); diff != "" {
				t.Errorf("parseRepositoriesConfig() mismatch (-want +got): %s", diff)
			}
		})
	}
}

func TestRepositoryGitUrl(t *testing.T) {
	for _, test := range []struct {
		name       string
		repository *RepositoryConfig
		want       string
		wantError  bool
	}{
		{
			name: "sets default organization",
			repository: &RepositoryConfig{
				Name: "google-cloud-python",
			},
			want: "https://github.com/googleapis/google-cloud-python",
		},
		{
			name: "reads full name",
			repository: &RepositoryConfig{
				FullName: "https://github.com/some-org/google-cloud-python",
			},
			want: "https://github.com/some-org/google-cloud-python",
		},
		{
			name: "prefers full name",
			repository: &RepositoryConfig{
				Name:     "google-cloud-python",
				FullName: "https://github.com/some-org/google-cloud-python",
			},
			want: "https://github.com/some-org/google-cloud-python",
		},
		{
			name: "missing name",
			repository: &RepositoryConfig{
				Name:     "google-cloud-python",
				FullName: "https://github.com/some-org/google-cloud-python",
			},
			want:      "https://github.com/some-org/google-cloud-python",
			wantError: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.repository.GitURL()
			if err != nil && test.wantError {
				t.Errorf("expected to return error")
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("parseRepositoriesConfig() mismatch (-want +got): %s", diff)
			}
		})
	}
}
