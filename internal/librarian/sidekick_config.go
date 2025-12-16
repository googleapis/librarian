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

package librarian

import (
	"github.com/googleapis/librarian/internal/config"
	sidekickconfig "github.com/googleapis/librarian/internal/sidekick/config"
)

// toSidekickReleaseConfig translates a librarian Release config to a sidekick
// Release config.
func toSidekickReleaseConfig(cfg *config.Release) *sidekickconfig.Release {
	if cfg == nil {
		return nil
	}
	tools := make(map[string][]sidekickconfig.Tool)
	for k, v := range cfg.Tools {
		var sidekickTools []sidekickconfig.Tool
		for _, t := range v {
			sidekickTools = append(sidekickTools, sidekickconfig.Tool{
				Name:    t.Name,
				Version: t.Version,
			})
		}
		tools[k] = sidekickTools
	}
	return &sidekickconfig.Release{
		Remote:         cfg.Remote,
		Branch:         cfg.Branch,
		Tools:          tools,
		Preinstalled:   cfg.Preinstalled,
		IgnoredChanges: cfg.IgnoredChanges,
		RootsPem:       cfg.RootsPem,
	}
}
