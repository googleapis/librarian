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

package dart

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
)

// Publish publishes the specified libraries in topological order.
func Publish(ctx context.Context, cfg *config.Config, librariesToPublish []string, execute bool) error {
	libraryByName := make(map[string]*config.Library)
	for _, lib := range cfg.Libraries {
		if lib.SkipRelease {
			continue
		}
		libraryByName[lib.Name] = lib
	}

	adj := make(map[string][]string)
	inDegree := make(map[string]int)
	for name := range libraryByName {
		inDegree[name] = 0
	}

	for _, lib := range cfg.Libraries {
		if lib.SkipRelease {
			continue
		}
		if lib.Dart == nil || lib.Dart.Packages == nil {
			continue
		}
		for dep := range lib.Dart.Packages {
			depName := strings.TrimPrefix(dep, "package:")
			if _, ok := libraryByName[depName]; ok {
				adj[depName] = append(adj[depName], lib.Name)
				inDegree[lib.Name]++
			}
		}
	}

	var queue []string
	for name, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}
	slices.Sort(queue)

	var topologicalOrder []string
	for len(queue) > 0 {
		currName := queue[0]
		queue = queue[1:]
		topologicalOrder = append(topologicalOrder, currName)

		for _, dep := range adj[currName] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
		slices.Sort(queue)
	}

	if len(topologicalOrder) < len(libraryByName) {
		return fmt.Errorf("cycle detected in dependency DAG")
	}

	// Filter topological order to keep only librariesToPublish
	toPublishSet := make(map[string]bool)
	for _, name := range librariesToPublish {
		toPublishSet[name] = true
	}

	var publishOrder []string
	for _, name := range topologicalOrder {
		if toPublishSet[name] {
			publishOrder = append(publishOrder, name)
		}
	}

	for _, name := range publishOrder {
		lib := libraryByName[name]
		output := libraryOutput(lib, cfg.Default)
		var args []string
		if execute {
			args = []string{"pub", "publish", "--force"}
		} else {
			args = []string{"pub", "publish", "--dry-run"}
		}

		if err := command.RunInDir(ctx, output, "dart", args...); err != nil {
			return fmt.Errorf("failed to publish library %s: %w", name, err)
		}
	}

	return nil
}
