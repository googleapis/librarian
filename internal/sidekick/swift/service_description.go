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
	"fmt"
	"regexp"
	"strings"
)

var reDocXref = regexp.MustCompile(`\[([^\]]+)\](?:\([^\)]+\)|\[[^\]]*\])`)

// extractServiceDescription extracts a concise single-line description from
// service documentation comments.
//
// A naive strategy of extracting the first paragraph in full fails in several
// common cases across Google Cloud APIs:
//  1. Title-only first paragraphs: Many services (such as SecretManagerService
//     and EkmServiceClient) begin with a title line that lacks sentence-ending
//     punctuation (e.g. "Secret Manager Service" or "Google Cloud Key Management
//     EKM Service"). Using the first paragraph merely repeats the client name
//     without explaining what the service does. When this pattern is detected,
//     the function skips the title line and extracts from the following paragraph.
//  2. Multi-sentence or overly detailed paragraphs: Other services (such as
//     AutoKeyClient) start with a long paragraph detailing implementation
//     mechanics, architecture, and resource associations. Extracting the full
//     paragraph clutters README and landing page overviews. Extracting only the
//     first sentence yields a punchy, high-level summary.
//  3. Proto cross-reference links: Comments often embed reference-style links
//     such as `[CryptoKey][google.cloud.kms.v1.CryptoKey]`. These reference
//     definitions are not defined in the overview page, leaving broken or ugly
//     links unless the targets are stripped to display text.
//  4. Section headers: Some services start with generic section headers such
//     as "API Overview:" which should be skipped.
func extractServiceDescription(doc string, serviceName string) string {
	if doc == "" {
		return fmt.Sprintf("Client for the %s.", serviceName)
	}
	paragraphs := strings.Split(strings.ReplaceAll(doc, "\r\n", "\n"), "\n\n")
	var cleaned []string
	for _, p := range paragraphs {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		// Skip short generic header lines ending in ":" like "API Overview:"
		if strings.HasSuffix(trimmed, ":") && len(trimmed) < 40 && !strings.Contains(trimmed, ". ") {
			continue
		}
		cleaned = append(cleaned, trimmed)
	}
	if len(cleaned) == 0 {
		return fmt.Sprintf("Client for the %s.", serviceName)
	}

	selected := cleaned[0]
	// If the first paragraph is just a title without sentence-ending punctuation,
	// and a subsequent paragraph exists, prefer the subsequent paragraph.
	if !strings.HasSuffix(selected, ".") && !strings.HasSuffix(selected, "!") && !strings.HasSuffix(selected, "?") && len(cleaned) > 1 {
		selected = cleaned[1]
	}

	// Remove proto cross-reference link targets like [Text][proto.id] -> Text
	// and markdown link targets like [Text](url) -> Text
	selected = reDocXref.ReplaceAllString(selected, "$1")
	selected = strings.ReplaceAll(selected, "`", "")
	fields := strings.Fields(selected)
	text := strings.Join(fields, " ")

	// Extract the first sentence if multiple sentences exist.
	if idx := strings.Index(text, ". "); idx != -1 {
		// If the first sentence is very short (< 30 chars), try to include the second sentence
		if idx < 30 {
			if secIdx := strings.Index(text[idx+2:], ". "); secIdx != -1 {
				text = text[:idx+2+secIdx+1]
			}
		} else {
			text = text[:idx+1]
		}
	} else if !strings.HasSuffix(text, ".") && !strings.HasSuffix(text, "!") && !strings.HasSuffix(text, "?") {
		text = strings.TrimSuffix(text, ":")
		text = strings.TrimSpace(text) + "."
	}

	if text == "" || text == "." {
		return fmt.Sprintf("Client for the %s.", serviceName)
	}
	return text
}
