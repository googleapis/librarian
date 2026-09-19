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
	for _, test := range []struct {
		name string
		pkg  string
		id   string
		want string
	}{
		{
			name: "empty id returns empty",
			pkg:  "google.cloud.example.v1",
			id:   "",
			want: "",
		},
		{
			name: "already fully qualified with leading dot",
			pkg:  "google.cloud.example.v1",
			id:   ".google.protobuf.Empty",
			want: ".google.protobuf.Empty",
		},
		{
			name: "has package without leading dot",
			pkg:  "google.cloud.example.v1",
			id:   "google.protobuf.Empty",
			want: ".google.protobuf.Empty",
		},
		{
			name: "bare symbol name with package",
			pkg:  "google.cloud.example.v1",
			id:   "MyResponse",
			want: ".google.cloud.example.v1.MyResponse",
		},
		{
			name: "bare symbol name with empty package",
			pkg:  "",
			id:   "MyResponse",
			want: ".MyResponse",
		},
		{
			name: "empty package and empty id",
			pkg:  "",
			id:   "",
			want: "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := normalizeTypeID(test.pkg, test.id)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
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

	for _, test := range []struct {
		name         string
		responseType string
		metadataType string
		want         *api.OperationInfo
	}{
		{
			name:         "method with valid response and metadata types",
			responseType: "Item",
			metadataType: "CreateItemMetadata",
			want: &api.OperationInfo{
				ResponseTypeID: ".test.pkg.Item",
				MetadataTypeID: ".test.pkg.CreateItemMetadata",
			},
		},
		{
			name:         "empty metadata type defaults to google.protobuf.Empty and avoids synthesizing invalid package dot",
			responseType: "google.protobuf.Empty",
			metadataType: "",
			want: &api.OperationInfo{
				ResponseTypeID: ".google.protobuf.Empty",
				MetadataTypeID: ".google.protobuf.Empty",
			},
		},
		{
			name:         "empty response type defaults to google.protobuf.Empty",
			responseType: "",
			metadataType: "PollMetadata",
			want: &api.OperationInfo{
				ResponseTypeID: ".google.protobuf.Empty",
				MetadataTypeID: ".test.pkg.PollMetadata",
			},
		},
		{
			name:         "both response and metadata empty returns nil to prevent incomplete OperationInfo",
			responseType: "",
			metadataType: "",
			want:         nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			m := &descriptorpb.MethodDescriptorProto{
				Name:    new("TestMethod"),
				Options: &descriptorpb.MethodOptions{},
			}
			proto.SetExtension(m.Options, longrunningpb.E_OperationInfo, &longrunningpb.OperationInfo{
				ResponseType: test.responseType,
				MetadataType: test.metadataType,
			})
			got := parseOperationInfo("test.pkg", m)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
