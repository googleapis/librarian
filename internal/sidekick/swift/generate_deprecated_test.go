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

package swift

import (
	"strings"
	"testing"
)

// checkDiagnose verifies whether the block suppressing deprecation warnings
// immediately precedes `decl`, a declaration indented by `indent`.
//
// The generator emits the block from `templates/common/diagnose.mustache` only
// where the generated code cannot avoid naming a deprecated declaration. Every
// occurrence of `decl` must agree, so a template guarded in one place but not
// another fails.
//
// That all-or-nothing contract makes `decl` load bearing. `"public func get("`
// matches the pass-through declarations, which take `DiagnoseTypes`, and the
// generated overloads, which take `DiagnoseFields`; a case where those two
// flags differ needs a prefix long enough to tell them apart, such as
// `"public func get(\n  name: Swift.String,"`.
//
// The expected block is the raw template output. `Generate` does not run
// `swift-format`, which in real generation indents the attribute one level
// further inside the `#if`.
func checkDiagnose(t *testing.T, content, indent, decl string, want bool) {
	t.Helper()
	declarations := strings.Count(content, "\n"+indent+decl)
	if declarations == 0 {
		t.Fatalf("missing declaration %q\n\n%s", decl, content)
	}
	block := "\n" + indent + "#if hasAttribute(diagnose)\n" +
		indent + "@diagnose(DeprecatedDeclaration, as: ignored)\n" +
		indent + "#endif\n" +
		indent + decl
	guarded := strings.Count(content, block)
	wantGuarded := 0
	if want {
		wantGuarded = declarations
	}
	if guarded != wantGuarded {
		t.Errorf("declarations of %q preceded by @diagnose: got %d of %d, want %d\n\n%s",
			decl, guarded, declarations, wantGuarded, content)
	}
}
