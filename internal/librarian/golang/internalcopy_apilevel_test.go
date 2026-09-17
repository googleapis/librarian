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

package golang

import (
	"errors"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/gofeaturespb"
)

func TestCheckInternalCopyAPILevel(t *testing.T) {
	for _, test := range []struct {
		name          string
		protoAPILevel string
		edition       descriptorpb.Edition
		fileLevel     gofeaturespb.GoFeatures_APILevel
		messageLevel  gofeaturespb.GoFeatures_APILevel
		nestedLevel   gofeaturespb.GoFeatures_APILevel
	}{
		{name: "default"},
		{name: "proto_api_level open", protoAPILevel: "API_OPEN"},
		{name: "file feature open overrides opaque proto_api_level", protoAPILevel: "API_OPAQUE", edition: descriptorpb.Edition_EDITION_2023, fileLevel: gofeaturespb.GoFeatures_API_OPEN},
		{name: "edition 2023 defaults to open", edition: descriptorpb.Edition_EDITION_2023},
		{name: "edition 2024 with open file feature", edition: descriptorpb.Edition_EDITION_2024, fileLevel: gofeaturespb.GoFeatures_API_OPEN},
		{name: "open message feature", edition: descriptorpb.Edition_EDITION_2023, messageLevel: gofeaturespb.GoFeatures_API_OPEN, nestedLevel: gofeaturespb.GoFeatures_API_OPEN},
	} {
		t.Run(test.name, func(t *testing.T) {
			fds := apiLevelDescriptorSet(test.edition, test.fileLevel, test.messageLevel, test.nestedLevel)
			if err := checkInternalCopyAPILevel(fds, []string{"foo/v1/a.proto"}, test.protoAPILevel); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestCheckInternalCopyAPILevel_Error(t *testing.T) {
	for _, test := range []struct {
		name          string
		protoAPILevel string
		apiFiles      []string
		edition       descriptorpb.Edition
		fileLevel     gofeaturespb.GoFeatures_APILevel
		messageLevel  gofeaturespb.GoFeatures_APILevel
		nestedLevel   gofeaturespb.GoFeatures_APILevel
		wantErr       error
		wantMsg       []string
	}{
		{
			name:          "proto_api_level opaque",
			protoAPILevel: "API_OPAQUE",
			wantErr:       errInternalCopyAPILevel,
			wantMsg:       []string{"file foo/v1/a.proto", "API_OPAQUE", "proto_api_level"},
		},
		{
			name:          "proto_api_level hybrid",
			protoAPILevel: "API_HYBRID",
			wantErr:       errInternalCopyAPILevel,
			wantMsg:       []string{"file foo/v1/a.proto", "API_HYBRID", "proto_api_level"},
		},
		{
			name:          "unknown proto_api_level",
			protoAPILevel: "API_FAST",
			wantErr:       errInternalCopyAPILevel,
			wantMsg:       []string{"API_FAST"},
		},
		{
			name:          "file feature opaque overrides open proto_api_level",
			protoAPILevel: "API_OPEN",
			edition:       descriptorpb.Edition_EDITION_2023,
			fileLevel:     gofeaturespb.GoFeatures_API_OPAQUE,
			wantErr:       errInternalCopyAPILevel,
			wantMsg:       []string{"file foo/v1/a.proto", "API_OPAQUE", "the file's api_level feature"},
		},
		{
			name:      "file feature hybrid",
			edition:   descriptorpb.Edition_EDITION_2023,
			fileLevel: gofeaturespb.GoFeatures_API_HYBRID,
			wantErr:   errInternalCopyAPILevel,
			wantMsg:   []string{"API_HYBRID"},
		},
		{
			name:          "edition 2024 defaults to opaque",
			protoAPILevel: "API_OPEN",
			edition:       descriptorpb.Edition_EDITION_2024,
			wantErr:       errInternalCopyAPILevel,
			wantMsg:       []string{"API_OPAQUE", "the default of EDITION_2024"},
		},
		{
			name:         "message feature opaque",
			edition:      descriptorpb.Edition_EDITION_2023,
			messageLevel: gofeaturespb.GoFeatures_API_OPAQUE,
			wantErr:      errInternalCopyAPILevel,
			wantMsg:      []string{"message foo.v1.A in foo/v1/a.proto", "the message's api_level feature"},
		},
		{
			name:        "nested message feature hybrid",
			edition:     descriptorpb.Edition_EDITION_2023,
			nestedLevel: gofeaturespb.GoFeatures_API_HYBRID,
			wantErr:     errInternalCopyAPILevel,
			wantMsg:     []string{"message foo.v1.A.Inner in foo/v1/a.proto", "API_HYBRID"},
		},
		{
			name:     "api file not in set",
			apiFiles: []string{"foo/v1/missing.proto"},
			wantErr:  errInternalCopyFileNotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fds := apiLevelDescriptorSet(test.edition, test.fileLevel, test.messageLevel, test.nestedLevel)
			apiFiles := test.apiFiles
			if apiFiles == nil {
				apiFiles = []string{"foo/v1/a.proto"}
			}
			gotErr := checkInternalCopyAPILevel(fds, apiFiles, test.protoAPILevel)
			if !errors.Is(gotErr, test.wantErr) {
				t.Fatalf("checkInternalCopyAPILevel error = %v, wantErr %v", gotErr, test.wantErr)
			}
			for _, want := range test.wantMsg {
				if !strings.Contains(gotErr.Error(), want) {
					t.Errorf("error %q does not mention %q", gotErr, want)
				}
			}
		})
	}
}

// apiLevelDescriptorSet builds a descriptor set with one API file holding a
// message with a nested message, with the given edition and api_level
// features. An unspecified level leaves the feature unset.
func apiLevelDescriptorSet(edition descriptorpb.Edition, fileLevel, messageLevel, nestedLevel gofeaturespb.GoFeatures_APILevel) *descriptorpb.FileDescriptorSet {
	fd := &descriptorpb.FileDescriptorProto{
		Name:    new("foo/v1/a.proto"),
		Package: new("foo.v1"),
		MessageType: []*descriptorpb.DescriptorProto{{
			Name:       new("A"),
			Options:    &descriptorpb.MessageOptions{Features: apiLevelFeatures(messageLevel)},
			NestedType: []*descriptorpb.DescriptorProto{{Name: new("Inner"), Options: &descriptorpb.MessageOptions{Features: apiLevelFeatures(nestedLevel)}}},
		}},
	}
	if edition != descriptorpb.Edition_EDITION_UNKNOWN {
		fd.Syntax = new("editions")
		fd.Edition = edition.Enum()
	}
	if features := apiLevelFeatures(fileLevel); features != nil {
		fd.Options = &descriptorpb.FileOptions{Features: features}
	}
	return &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{fd}}
}

func apiLevelFeatures(level gofeaturespb.GoFeatures_APILevel) *descriptorpb.FeatureSet {
	if level == gofeaturespb.GoFeatures_API_LEVEL_UNSPECIFIED {
		return nil
	}
	features := &descriptorpb.FeatureSet{}
	proto.SetExtension(features, gofeaturespb.E_Go, &gofeaturespb.GoFeatures{ApiLevel: level.Enum()})
	return features
}
