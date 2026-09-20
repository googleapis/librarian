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

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestParseOperationService(t *testing.T) {
	for _, test := range []struct {
		name   string
		method *descriptorpb.MethodDescriptorProto
		want   string
	}{
		{
			name:   "nil method descriptor",
			method: nil,
			want:   "",
		},
		{
			name: "no options",
			method: &descriptorpb.MethodDescriptorProto{
				Name: new("NoOptionsMethod"),
			},
			want: "",
		},
		{
			name: "options without extension",
			method: &descriptorpb.MethodDescriptorProto{
				Name:    new("NoExtensionMethod"),
				Options: &descriptorpb.MethodOptions{},
			},
			want: "",
		},
		{
			name:   "extension set to RegionOperations",
			method: newTestMethodWithOperationService("RegionMethod", "RegionOperations"),
			want:   "RegionOperations",
		},
		{
			name:   "extension set to GlobalOperations",
			method: newTestMethodWithOperationService("GlobalMethod", "GlobalOperations"),
			want:   "GlobalOperations",
		},
		{
			name:   "extension set to ZoneOperations",
			method: newTestMethodWithOperationService("ZoneMethod", "ZoneOperations"),
			want:   "ZoneOperations",
		},
		{
			name:   "extension set to empty string",
			method: newTestMethodWithOperationService("EmptyMethod", ""),
			want:   "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := parseOperationService(test.method)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestProcessMethod_OperationService(t *testing.T) {
	for _, test := range []struct {
		name   string
		method *descriptorpb.MethodDescriptorProto
		want   string
	}{
		{
			name: "no options",
			method: &descriptorpb.MethodDescriptorProto{
				Name: new("Insert"),
			},
			want: "",
		},
		{
			name: "options without extension",
			method: &descriptorpb.MethodDescriptorProto{
				Name:    new("Delete"),
				Options: &descriptorpb.MethodOptions{},
			},
			want: "",
		},
		{
			name:   "extension set to RegionOperations",
			method: newTestMethodWithOperationService("InsertRegion", "RegionOperations"),
			want:   "RegionOperations",
		},
		{
			name:   "extension set to GlobalOperations",
			method: newTestMethodWithOperationService("SetIamPolicyGlobal", "GlobalOperations"),
			want:   "GlobalOperations",
		},
		{
			name:   "extension set to ZoneOperations",
			method: newTestMethodWithOperationService("AttachDiskZone", "ZoneOperations"),
			want:   "ZoneOperations",
		},
		{
			name:   "extension set to empty string",
			method: newTestMethodWithOperationService("EmptyMethod", ""),
			want:   "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			model := api.NewTestAPI(nil, nil, nil)
			mFQN := ".test.service.Method"
			got, err := processMethod(model, test.method, mFQN, "test.service", "test.service.Service", "v1")
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got.OperationService); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			modelMethod := model.Method(mFQN)
			if modelMethod == nil {
				t.Fatalf("expected method %q to be added to model", mFQN)
			}
			if diff := cmp.Diff(test.want, modelMethod.OperationService); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func newTestMethodWithOperationService(name, opService string) *descriptorpb.MethodDescriptorProto {
	m := &descriptorpb.MethodDescriptorProto{
		Name:    new(name),
		Options: &descriptorpb.MethodOptions{},
	}
	proto.SetExtension(m.Options, eOperationService, opService)
	return m
}
