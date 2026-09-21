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
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestGenerateMessage_Deprecated(t *testing.T) {
	for _, test := range []struct {
		name             string
		topDeprecated    bool
		nestedDeprecated bool
		wantTop          string
		wantNested       string
	}{
		{
			name:             "deprecated-both",
			topDeprecated:    true,
			nestedDeprecated: true,
			wantTop:          "/// -- top marker --\n@available(*, deprecated)\npublic struct TopMessage",
			wantNested:       "  /// -- nested marker --\n  @available(*, deprecated)\n  public struct NestedMessage",
		},
		{
			name:             "deprecated-top-only",
			topDeprecated:    true,
			nestedDeprecated: false,
			wantTop:          "/// -- top marker --\n@available(*, deprecated)\npublic struct TopMessage",
			wantNested:       "  /// -- nested marker --\n  public struct NestedMessage",
		},
		{
			name:             "deprecated-nested-only",
			topDeprecated:    false,
			nestedDeprecated: true,
			wantTop:          "/// -- top marker --\npublic struct TopMessage",
			wantNested:       "  /// -- nested marker --\n  @available(*, deprecated)\n  public struct NestedMessage",
		},
		{
			name:             "not-deprecated",
			topDeprecated:    false,
			nestedDeprecated: false,
			wantTop:          "/// -- top marker --\npublic struct TopMessage",
			wantNested:       "  /// -- nested marker --\n  public struct NestedMessage",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()

			nested := &api.Message{
				Name:          "NestedMessage",
				Package:       "google.cloud.test.v1",
				ID:            ".google.cloud.test.v1.TopMessage.NestedMessage",
				Deprecated:    test.nestedDeprecated,
				Documentation: "-- nested marker --",
			}

			top := &api.Message{
				Name:          "TopMessage",
				Package:       "google.cloud.test.v1",
				ID:            ".google.cloud.test.v1.TopMessage",
				Deprecated:    test.topDeprecated,
				Documentation: "-- top marker --",
				Messages:      []*api.Message{nested},
			}

			model := api.NewTestAPI([]*api.Message{top}, nil, nil)
			model.PackageName = "google.cloud.test.v1"
			if err := Generate(t.Context(), model, outDir, &config.Library{}, nil); err != nil {
				t.Fatal(err)
			}

			filename := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "TopMessage.swift")
			content, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			contentStr := string(content)

			gotTop := extractBlock(t, contentStr, "/// -- top marker --", "public struct TopMessage")
			if diff := cmp.Diff(test.wantTop, gotTop); diff != "" {
				t.Errorf("mismatch top (-want +got):\n%s", diff)
			}

			gotNested := extractBlock(t, contentStr, "  /// -- nested marker --", "public struct NestedMessage")
			if diff := cmp.Diff(test.wantNested, gotNested); diff != "" {
				t.Errorf("mismatch nested (-want +got):\n%s", diff)
			}
		})
	}
}

// TestGenerateMessage_Diagnose covers the members that read and write every
// field, and therefore name each deprecated field and field type.
func TestGenerateMessage_Diagnose(t *testing.T) {
	for _, test := range []struct {
		name              string
		messageDeprecated bool
		fieldDeprecated   bool
		typeDeprecated    bool
		want              bool
	}{
		{
			name:            "deprecated-field",
			fieldDeprecated: true,
			want:            true,
		},
		{
			name:           "deprecated-field-type",
			typeDeprecated: true,
			want:           true,
		},
		{
			// Swift does not diagnose deprecated references inside a
			// deprecated declaration.
			name:              "deprecated-message",
			messageDeprecated: true,
			fieldDeprecated:   true,
			typeDeprecated:    true,
			want:              false,
		},
		{
			name: "not-deprecated",
			want: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()

			other := api.NewTestMessage("OtherMessage").
				WithPackage("google.cloud.test.v1").
				WithDeprecated(test.typeDeprecated)
			msg := api.NewTestMessage("TestMessage").
				WithPackage("google.cloud.test.v1").
				WithDeprecated(test.messageDeprecated).
				WithFields(
					api.NewTestField("normal_field").
						WithType(api.TypezString).
						WithDeprecated(test.fieldDeprecated),
					api.NewTestField("other_field").WithMessageType(other),
				)

			model := api.NewTestAPI([]*api.Message{msg, other}, nil, nil)
			model.PackageName = "google.cloud.test.v1"
			if err := Generate(t.Context(), model, outDir, &config.Library{}, nil); err != nil {
				t.Fatal(err)
			}

			filename := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "TestMessage.swift")
			content, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			contentStr := string(content)

			checkDiagnose(t, contentStr, "  ", "public init(from decoder: Decoder) throws {", test.want)
			checkDiagnose(t, contentStr, "  ", "public func encode(to encoder: Encoder) throws {", test.want)
		})
	}
}

// TestGenerateMessage_DiagnosePagination covers the
// `GoogleGax._PaginatedResponse` members.
//
// `_getPaginatedItems()` names the item type in its return type, so guarding
// the property that holds the items is not enough.
func TestGenerateMessage_DiagnosePagination(t *testing.T) {
	for _, test := range []struct {
		name                string
		messageDeprecated   bool
		itemFieldDeprecated bool
		itemTypeDeprecated  bool
		tokenDeprecated     bool
		want                bool
	}{
		{
			name:               "deprecated-item-type",
			itemTypeDeprecated: true,
			want:               true,
		},
		{
			name:                "deprecated-item-field",
			itemFieldDeprecated: true,
			want:                true,
		},
		{
			name:            "deprecated-page-token",
			tokenDeprecated: true,
			want:            true,
		},
		{
			// Swift does not diagnose deprecated references inside a
			// deprecated declaration.
			name:               "deprecated-message",
			messageDeprecated:  true,
			itemTypeDeprecated: true,
			want:               false,
		},
		{
			name: "not-deprecated",
			want: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()

			item := api.NewTestMessage("Item").
				WithPackage("google.cloud.test.v1").
				WithDeprecated(test.itemTypeDeprecated)
			itemField := api.NewTestField("items").
				WithMessageType(item).
				WithRepeated().
				WithDeprecated(test.itemFieldDeprecated)
			nextPageToken := api.NewTestField("next_page_token").
				WithType(api.TypezString).
				WithDeprecated(test.tokenDeprecated)
			response := api.NewTestMessage("ListItemsResponse").
				WithPackage("google.cloud.test.v1").
				WithDeprecated(test.messageDeprecated).
				WithFields(itemField, nextPageToken).
				WithPagination(nextPageToken, itemField)

			model := api.NewTestAPI([]*api.Message{response, item}, nil, nil)
			model.PackageName = "google.cloud.test.v1"
			library := &config.Library{
				Swift: swiftConfig(t, []config.SwiftDependency{
					{Name: "GoogleGax", RequiredByServices: true},
				}),
			}
			if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
				t.Fatal(err)
			}

			filename := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "ListItemsResponse.swift")
			content, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			contentStr := string(content)

			checkDiagnose(t, contentStr, "  ", "public func _getPaginatedItems()", test.want)
			checkDiagnose(t, contentStr, "  ", "public func _nextPageToken()", test.want)
		})
	}
}
