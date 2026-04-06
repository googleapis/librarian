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

package nodejs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstall(t *testing.T) {
	// Pre-populate the fetch cache so fetch.Repo returns immediately
	// without downloading the tarball over the network.
	cache := t.TempDir()
	t.Setenv("LIBRARIAN_CACHE", cache)
	genDir := filepath.Join(cache,
		"github.com/googleapis/google-cloud-node@2ac5cf7a0dfb759be33ce24a40aae5b543ee375c",
		gapicGeneratorSubdir)
	for _, sub := range []string{"templates", "protos"} {
		if err := os.MkdirAll(filepath.Join(genDir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Stub npm so "npm install" and "npm link" are no-ops. The npm stub
	// also creates node_modules/.bin/tsc in the working directory so the
	// subsequent "./node_modules/.bin/tsc" build step finds an executable.
	bin := t.TempDir()
	npmStub := `#!/bin/sh
mkdir -p node_modules/.bin
printf '#!/bin/sh\nmkdir -p build\n' > node_modules/.bin/tsc
chmod +x node_modules/.bin/tsc
`
	if err := os.WriteFile(filepath.Join(bin, "npm"), []byte(npmStub), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "pip"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	if err := Install(t.Context()); err != nil {
		t.Fatal(err)
	}
}
