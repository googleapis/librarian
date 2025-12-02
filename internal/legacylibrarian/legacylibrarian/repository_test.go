// Copyright 2025 Google LLC
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

package legacylibrarian

import (
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	gogitConfig "github.com/go-git/go-git/v5/config"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacygithub"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacygitrepo"
)

func TestGetGitHubRepositoryFromGitRepo(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name          string
		remotes       map[string][]string
		wantRepo      *legacygithub.Repository
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name: "origin is a GitHub remote",
			remotes: map[string][]string{
				"origin": {"https://github.com/owner/repo.git"},
			},
			wantRepo: &legacygithub.Repository{Owner: "owner", Name: "repo"},
		},
		{
			name:          "No remotes",
			remotes:       map[string][]string{},
			wantErr:       true,
			wantErrSubstr: "could not find an 'origin' remote",
		},
		{
			name: "origin is not a GitHub remote",
			remotes: map[string][]string{
				"origin": {"https://gitlab.com/owner/repo.git"},
			},
			wantErr:       true,
			wantErrSubstr: "is not a GitHub remote",
		},
		{
			name: "upstream is GitHub, but no origin",
			remotes: map[string][]string{
				"gitlab":   {"https://gitlab.com/owner/repo.git"},
				"upstream": {"https://github.com/gh-owner/gh-repo.git"},
			},
			wantErr:       true,
			wantErrSubstr: "could not find an 'origin' remote",
		},
		{
			name: "origin and upstream are GitHub remotes, should use origin",
			remotes: map[string][]string{
				"origin":   {"https://github.com/owner/repo.git"},
				"upstream": {"https://github.com/owner2/repo2.git"},
			},
			wantRepo: &legacygithub.Repository{Owner: "owner", Name: "repo"},
		},
		{
			name: "origin is not GitHub, but upstream is",
			remotes: map[string][]string{
				"origin":   {"https://gitlab.com/owner/repo.git"},
				"upstream": {"https://github.com/gh-owner/gh-repo.git"},
			},
			wantErr:       true,
			wantErrSubstr: "is not a GitHub remote",
		},
		{
			name: "origin has multiple URLs, first is GitHub",
			remotes: map[string][]string{
				"origin": {"https://github.com/owner/repo.git", "https://gitlab.com/owner/repo.git"},
			},
			wantRepo: &legacygithub.Repository{Owner: "owner", Name: "repo"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			repo := newTestGitRepoWithRemotes(t, test.remotes)

			got, err := GetGitHubRepositoryFromGitRepo(repo)

			if test.wantErr {
				if err == nil {
					t.Fatalf("FetchGitHubRepoFromRemote() err = nil, want error containing %q", test.wantErrSubstr)
				}
				if !strings.Contains(err.Error(), test.wantErrSubstr) {
					t.Errorf("FetchGitHubRepoFromRemote() err = %v, want error containing %q", err, test.wantErrSubstr)
				}
			} else {
				if err != nil {
					t.Errorf("FetchGitHubRepoFromRemote() err = %v, want nil", err)
				}
				if diff := cmp.Diff(test.wantRepo, got); diff != "" {
					t.Errorf("FetchGitHubRepoFromRemote() repo mismatch (-want +got): %s", diff)
				}
			}
		})
	}
}

// newTestGitRepo creates a new git repository in a temporary directory with the given remotes.
func newTestGitRepoWithRemotes(t *testing.T, remotes map[string][]string) *legacygitrepo.LocalRepository {
	t.Helper()
	dir := t.TempDir()

	r, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("git.PlainInit failed: %v", err)
	}

	for name, urls := range remotes {
		_, err := r.CreateRemote(&gogitConfig.RemoteConfig{
			Name: name,
			URLs: urls,
		})
		if err != nil {
			t.Fatalf("CreateRemote failed: %v", err)
		}
	}

	repo, err := legacygitrepo.NewRepository(&legacygitrepo.RepositoryOptions{Dir: dir})
	if err != nil {
		t.Fatalf("legacygitrepo.NewRepository failed: %v", err)
	}
	return repo
}
