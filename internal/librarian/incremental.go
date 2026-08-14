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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/proto"
	"github.com/googleapis/librarian/internal/sources"
)

// generatedManifestName is written into a library's output directory to record
// the fingerprint of the inputs that produced it, enabling --changed-only to
// skip regenerating unchanged libraries.
const generatedManifestName = ".librarian-inputs"

// fingerprintFiles hashes the contents of files (order-independent) together
// with extra. Returns a hex digest.
func fingerprintFiles(files []string, extra string) (string, error) {
	sorted := append([]string(nil), files...)
	sort.Strings(sorted)
	h := sha256.New()
	fmt.Fprintf(h, "v1\x00%s\n", extra)
	for _, f := range sorted {
		data, err := os.ReadFile(f)
		if err != nil {
			return "", fmt.Errorf("failed to read %q for fingerprint: %w", f, err)
		}
		fmt.Fprintf(h, "%d\x00", len(data))
		h.Write(data)
		h.Write([]byte{'\n'})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// configExtra captures generation-affecting configuration outside a library's
// protos — the language and tool versions — so a generator or toolchain change
// invalidates every manifest.
func configExtra(cfg *config.Config) string {
	b, err := json.Marshal(struct {
		Language string        `json:"language"`
		Tools    *config.Tools `json:"tools"`
	}{Language: cfg.Language, Tools: cfg.Tools})
	if err != nil {
		return cfg.Language
	}
	return string(b)
}

// libraryInputFingerprint fingerprints a library's generation inputs: the proto
// files under each of its API paths in the primary source root, plus the
// library config and the supplied extra. It mirrors how the Java generator
// resolves its inputs.
func libraryInputFingerprint(lib *config.Library, src *sources.Sources, extra string) (string, error) {
	if src == nil {
		return "", errors.New("no sources available")
	}
	srcCfg := sources.NewSourceConfig(src, lib.Roots)
	if len(srcCfg.ActiveRoots) == 0 {
		return "", errors.New("no active source roots")
	}
	primaryDir := srcCfg.Root(srcCfg.ActiveRoots[0])
	var files []string
	for _, api := range lib.APIs {
		protos, err := proto.Gather(filepath.Join(primaryDir, api.Path), api.Path)
		if err != nil {
			return "", fmt.Errorf("gather protos for %q: %w", api.Path, err)
		}
		files = append(files, protos...)
	}
	libJSON, err := json.Marshal(lib)
	if err != nil {
		return "", err
	}
	return fingerprintFiles(files, extra+"\x00"+string(libJSON))
}

func generatedManifestPath(outputDir string) string {
	return filepath.Join(outputDir, generatedManifestName)
}

// generatedUpToDate reports whether outputDir already holds a generation whose
// recorded input fingerprint matches fingerprint.
func generatedUpToDate(outputDir, fingerprint string) bool {
	if fingerprint == "" {
		return false
	}
	data, err := os.ReadFile(generatedManifestPath(outputDir))
	return err == nil && strings.TrimSpace(string(data)) == fingerprint
}

// writeGeneratedManifest records fingerprint in outputDir after a successful
// generation.
func writeGeneratedManifest(outputDir, fingerprint string) error {
	if fingerprint == "" {
		return nil
	}
	return os.WriteFile(generatedManifestPath(outputDir), []byte(fingerprint), 0o644)
}

// filterChangedLibraries partitions libraries for --changed-only: it returns the
// libraries whose inputs differ from their recorded manifest (plus any it cannot
// fingerprint, which are regenerated to be safe) and a map of library name to
// the freshly computed fingerprint for those kept.
func filterChangedLibraries(libraries []*config.Library, src *sources.Sources, extra string) ([]*config.Library, map[string]string) {
	kept := make([]*config.Library, 0, len(libraries))
	fingerprints := make(map[string]string, len(libraries))
	for _, lib := range libraries {
		fp, err := libraryInputFingerprint(lib, src, extra)
		if err != nil {
			slog.Debug("changed-only: cannot fingerprint, will regenerate", "library", lib.Name, "err", err)
			kept = append(kept, lib)
			continue
		}
		if generatedUpToDate(lib.Output, fp) {
			slog.Info("changed-only: inputs unchanged, skipping", "library", lib.Name)
			continue
		}
		kept = append(kept, lib)
		fingerprints[lib.Name] = fp
	}
	return kept, fingerprints
}
