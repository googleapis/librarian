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

package parser

import (
	"testing"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestNormalizeTypeID(t *testing.T) {
	tests := []struct {
		name     string
		pkg      string
		id       string
		expected string
	}{
		{
			name:     "empty id returns empty",
			pkg:      "google.cloud.example.v1",
			id:       "",
			expected: "",
		},
		{
			name:     "already fully qualified with leading dot",
			pkg:      "google.cloud.example.v1",
			id:       ".google.protobuf.Empty",
			expected: ".google.protobuf.Empty",
		},
		{
			name:     "has package without leading dot",
			pkg:      "google.cloud.example.v1",
			id:       "google.protobuf.Empty",
			expected: ".google.protobuf.Empty",
		},
		{
			name:     "bare symbol name with package",
			pkg:      "google.cloud.example.v1",
			id:       "MyResponse",
			expected: ".google.cloud.example.v1.MyResponse",
		},
		{
			name:     "bare symbol name with empty package",
			pkg:      "",
			id:       "MyResponse",
			expected: ".MyResponse",
		},
		{
			name:     "empty package and empty id",
			pkg:      "",
			id:       "",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeTypeID(tc.pkg, tc.id)
			if got != tc.expected {
				t.Errorf("normalizeTypeID(%q, %q) = %q, want %q", tc.pkg, tc.id, got, tc.expected)
			}
		})
	}
}

func TestParseOperationInfo_SafeParsing(t *testing.T) {
	t.Run("method without OperationInfo extension", func(t *testing.T) {
		m := &descriptorpb.MethodDescriptorProto{
			Name: new("NonLroMethod"),
		}
		got := parseOperationInfo("test.pkg", m)
		if got != nil {
			t.Errorf("expected nil for method without OperationInfo, got %+v", got)
		}
	})

	t.Run("method with valid response and metadata types", func(t *testing.T) {
		m := &descriptorpb.MethodDescriptorProto{
			Name:    new("CreateItem"),
			Options: &descriptorpb.MethodOptions{},
		}
		proto.SetExtension(m.Options, longrunningpb.E_OperationInfo, &longrunningpb.OperationInfo{
			ResponseType: "Item",
			MetadataType: "CreateItemMetadata",
		})

		got := parseOperationInfo("test.pkg", m)
		want := &api.OperationInfo{
			ResponseTypeID: ".test.pkg.Item",
			MetadataTypeID: ".test.pkg.CreateItemMetadata",
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("empty metadata type defaults to google.protobuf.Empty and avoids synthesizing invalid package dot", func(t *testing.T) {
		m := &descriptorpb.MethodDescriptorProto{
			Name:    new("DeleteItem"),
			Options: &descriptorpb.MethodOptions{},
		}
		proto.SetExtension(m.Options, longrunningpb.E_OperationInfo, &longrunningpb.OperationInfo{
			ResponseType: "google.protobuf.Empty",
			MetadataType: "",
		})

		got := parseOperationInfo("test.pkg", m)
		want := &api.OperationInfo{
			ResponseTypeID: ".google.protobuf.Empty",
			MetadataTypeID: ".google.protobuf.Empty",
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("empty response type defaults to google.protobuf.Empty", func(t *testing.T) {
		m := &descriptorpb.MethodDescriptorProto{
			Name:    new("PollItem"),
			Options: &descriptorpb.MethodOptions{},
		}
		proto.SetExtension(m.Options, longrunningpb.E_OperationInfo, &longrunningpb.OperationInfo{
			ResponseType: "",
			MetadataType: "PollMetadata",
		})

		got := parseOperationInfo("test.pkg", m)
		want := &api.OperationInfo{
			ResponseTypeID: ".google.protobuf.Empty",
			MetadataTypeID: ".test.pkg.PollMetadata",
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("both response and metadata empty returns nil to prevent incomplete OperationInfo", func(t *testing.T) {
		m := &descriptorpb.MethodDescriptorProto{
			Name:    new("InvalidLroMethod"),
			Options: &descriptorpb.MethodOptions{},
		}
		proto.SetExtension(m.Options, longrunningpb.E_OperationInfo, &longrunningpb.OperationInfo{
			ResponseType: "",
			MetadataType: "",
		})

		got := parseOperationInfo("test.pkg", m)
		if got != nil {
			t.Errorf("expected nil for empty OperationInfo, got %+v", got)
		}
	})
}
