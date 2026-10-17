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

package librarian

import (
	"slices"
	"strings"
	"testing"
)

const module = "github.com/googleapis/librarian/"

// leafPackages import nothing else from this module, which is what lets
// every other layer depend on them freely.
var leafPackages = []string{
	"internal/cache",
	"internal/command",
	"internal/license",
	"internal/proto",
	"internal/semver",
	"internal/sidekick/api",
	"internal/snippetmetadata",
	"internal/sources",
}

// dependencyRule forbids some imports from some packages. Paths are
// module-relative for from and full import paths for forbid.
type dependencyRule struct {
	name   string
	from   func(pkg string) bool
	forbid func(dep string) bool
}

// dependencyRules is the package layering of this repository.
var dependencyRules = []dependencyRule{
	{
		name:   "leaf packages import nothing from the module",
		from:   func(pkg string) bool { return slices.Contains(leafPackages, pkg) },
		forbid: func(dep string) bool { return strings.HasPrefix(dep, module) },
	},
	{
		name:   "internal/config is pure data and only imports internal/yaml",
		from:   func(pkg string) bool { return pkg == "internal/config" },
		forbid: func(dep string) bool { return strings.HasPrefix(dep, module) && dep != module+"internal/yaml" },
	},
	{
		name:   "the sidekick engine never imports the command layer",
		from:   func(pkg string) bool { return strings.HasPrefix(pkg, "internal/sidekick/") },
		forbid: func(dep string) bool { return strings.HasPrefix(dep, module+"internal/librarian") },
	},
	{
		name:   "language integrations are only imported by internal/librarian",
		from:   func(pkg string) bool { return pkg != "internal/librarian" },
		forbid: func(dep string) bool { return strings.HasPrefix(dep, module+"internal/librarian/") },
	},
	{
		name:   "only internal/yaml imports the YAML library",
		from:   func(pkg string) bool { return pkg != "internal/yaml" },
		forbid: func(dep string) bool { return strings.HasPrefix(dep, "go.yaml.in/yaml") },
	},
	{
		name:   "only internal/git imports go-git",
		from:   func(pkg string) bool { return pkg != "internal/git" },
		forbid: func(dep string) bool { return strings.HasPrefix(dep, "github.com/go-git/") },
	},
	{
		name:   "test helpers are not imported by production code",
		from:   func(pkg string) bool { return true },
		forbid: func(dep string) bool { return dep == module+"internal/testhelper" },
	},
	{
		name: "nothing imports cmd/ or tool/ packages",
		from: func(pkg string) bool { return true },
		forbid: func(dep string) bool {
			return strings.HasPrefix(dep, module+"cmd/") || strings.HasPrefix(dep, module+"tool/")
		},
	},
}

// TestDependencyRules enforces dependencyRules against the non-test imports
// of every package in the module.
func TestDependencyRules(t *testing.T) {
	imports := moduleImports(t)
	for _, rule := range dependencyRules {
		t.Run(rule.name, func(t *testing.T) {
			for pkg, deps := range imports {
				if !rule.from(pkg) {
					continue
				}
				for _, dep := range deps {
					if rule.forbid(dep) {
						t.Errorf("%s imports %s", pkg, dep)
					}
				}
			}
		})
	}
}

// moduleImports returns the non-test imports of every package in the module,
// keyed by module-relative import path.
func moduleImports(t *testing.T) map[string][]string {
	t.Helper()
	out := rungo(t, "list", "-f", `{{.ImportPath}} {{join .Imports " "}}`, "./...")
	imports := make(map[string][]string)
	for line := range strings.Lines(out) {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		imports[strings.TrimPrefix(fields[0], module)] = fields[1:]
	}
	return imports
}
