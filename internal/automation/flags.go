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
	"flag"

	"github.com/googleapis/librarian/internal/config"
)

func addFlagBuild(fs *flag.FlagSet, cfg *config.Config) {
	fs.BoolVar(&cfg.Build, "build", false,
		"The _BUILD flag (true/false) to Librarian CLI's -build option")
}

func addFlagCommand(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.CommandName, "command", "generate",
		"The librarian command to run")
}

func addFlagForceRun(fs *flag.FlagSet, cfg *config.Config) {
	fs.BoolVar(&cfg.ForceRun, "force-run", false,
		"The _FORCE_RUN flag (true/false) to Librarian CLI's -force-run option")
}

func addFlagProject(fs *flag.FlagSet, cfg *config.Config) {
	fs.StringVar(&cfg.Project, "project", "cloud-sdk-librarian-prod",
		"GCP project ID")
}

func addFlagPush(fs *flag.FlagSet, cfg *config.Config) {
	fs.BoolVar(&cfg.Push, "push", true,
		"The _PUSH flag (true/false) to Librarian CLI's -push option")
}
