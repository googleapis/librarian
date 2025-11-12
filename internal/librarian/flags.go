// Copyright 2024 Google LLC
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
	"flag"
	"fmt"

	"github.com/googleapis/librarian/internal/config"
)

func addFlagAPI(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.API, "api", "",
		`Relative path to the API to be configured/generated (e.g., google/cloud/functions/v2).
Must be specified when generating a new library.`)
}

func addFlagAPISource(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.APISource, "api-source", "https://github.com/googleapis/googleapis",
		`The location of an API specification repository.
Can be a remote URL or a local file path.`)
}

func addFlagBuild(fs *flag.FlagSet, cfg *config.Config) {
	fs.BoolVar(&cfg.Build, "build", false,
		`If true, Librarian will build each generated library by invoking the
language-specific container.`)
}

func addFlagBranch(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.Branch, "branch", "main",
		`The branch to use with remote code repositories. This is used to specify
which branch to clone and which branch to use as the base for a pull
request.`)
}

func addFlagCheckUnexpectedChanges(fs *flag.FlagSet, cfg *config.Config) {
	fs.BoolVar(&cfg.CheckUnexpectedChanges, "check-unexpected-changes", false,
		`Defaults to false. When used with --test, this flag verifies that no
unexpected files are added, deleted, or modified outside of the changes caused
by proto updates. You may want to skip this check when testing a container image
change that is expected to add or delete files.`)
}

func addFlagCommit(fs *flag.FlagSet, cfg *config.Config) {
	fs.BoolVar(&cfg.Commit, "commit", false,
		`If true, librarian will create a commit for the change but not create
a pull request. This flag is ignored if push is set to true.`)
}

func addFlagGenerateUnchanged(fs *flag.FlagSet, cfg *config.Config) {
	fs.BoolVar(&cfg.GenerateUnchanged, "generate-unchanged", false,
		`If true, librarian generates libraries even if none of their associated APIs
have changed. This does not override generation being blocked by configuration.`)
}

func addFlagGitHubAPIEndpoint(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.GitHubAPIEndpoint, "github-api-endpoint", "",
		`The GitHub API endpoint to use for all GitHub API operations.
This is intended for testing and should not be used in production.`)
}

func addFlagHostMount(fs *flag.FlagSet, cfg *config.Config) {
	defaultValue := ""
	fs.StringVar(&cfg.HostMount, "host-mount", defaultValue,
		`For use when librarian is running in a container. A mapping of a
directory from the host to the container, in the format
<host-mount>:<local-mount>.`)
}

func addFlagImage(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.Image, "image", "",
		`Language specific image used to invoke code generation and releasing.
If not specified, the image configured in the state.yaml is used.`)
}

func addFlagLibrary(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.Library, "library", "",
		`The library ID to generate or release (e.g. secretmanager).
This corresponds to a releasable language unit.`)
}

func addFlagLibraryToTest(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.LibraryToTest, "library-to-test", "",
		`When used with --test, this flag specifies the library ID to test
(e.g. secretmanager). Will test on all configured libraries if omitted.`)
}

func addFlagLibraryVersion(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.LibraryVersion, "library-version", "",
		`Overrides the automatic semantic version calculation and forces a specific
version for a library. Requires the --library flag to be specified.`)
}

func addFlagPR(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.PullRequest, "pr", "",
		`The URL of a pull request to operate on.
It should be in the format of https://github.com/{owner}/{repo}/pull/{number}.
If not specified, will search for all merged pull requests with the label
"release:pending" in the last 30 days.`)
}

func addFlagPush(fs *flag.FlagSet, cfg *config.Config) {
	fs.BoolVar(&cfg.Push, "push", false,
		fmt.Sprintf(`If true, Librarian will create a commit, 
push and create a pull request for the changes.
A GitHub token with push access must be provided via the
%s environment variable.`, config.LibrarianGithubToken))
}

func addFlagRepo(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.Repo, "repo", "",
		`Code repository where the generated code will reside. Can be a remote
in the format of a remote URL such as https://github.com/{owner}/{repo} or a
local file path like /path/to/repo. Both absolute and relative paths are
supported. If not specified, will try to detect if the current working directory
is configured as a language repository.
Note: When using a local repository (either by providing a path or by defaulting
to the current directory), Librarian creates a new branch from the currently
checked-out branch and commits changes there. If the -push flag is also specified,
a pull request is created against the main branch.`)
}

func addFlagTest(fs *flag.FlagSet, cfg *config.Config) {
	fs.BoolVar(&cfg.Test, "test", false,
		`If true, run container tests after generation but before committing and pushing.
These tests verify the interaction between language containers and the Librarian CLI's
'generate' command. If a test fails, temporary branches and files will be preserved for
debugging. This flag can be used with 'library-to-test' and 'check-unexpected-changes'.`)
}

func addFlagWorkRoot(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.WorkRoot, "output", "",
		`Working directory root. When this is not specified, a working directory
will be created in /tmp.`)
}

func addFlagVerbose(fs *flag.FlagSet, p *bool) {
	fs.BoolVar(p, "v", false, "enables verbose logging")
}
