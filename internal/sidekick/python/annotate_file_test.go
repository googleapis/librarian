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

package python

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateFile(t *testing.T) {
	for _, test := range []struct {
		name       string
		sourceFile string
		messages   []*api.Message
		enums      []*api.Enum
		want       *fileAnnotations
	}{
		{
			name:       "basic file with messages and enums",
			sourceFile: "google/cloud/secretmanager/v1/resources.proto",
			messages: []*api.Message{
				api.NewTestMessage("SecretPayload").
					WithSourceLocation("google/cloud/secretmanager/v1/resources.proto", 1),
				api.NewTestMessage("Secret").
					WithSourceLocation("google/cloud/secretmanager/v1/resources.proto", 2),
			},
			enums: []*api.Enum{
				api.NewTestEnum("SecretStatus").
					WithSourceLocation("google/cloud/secretmanager/v1/resources.proto", 3),
			},
			want: &fileAnnotations{
				SourceFile:         "google/cloud/secretmanager/v1/resources.proto",
				Stem:               "resources",
				HasMessagesOrEnums: true,
				Manifest:           []string{"SecretStatus", "SecretPayload", "Secret"},
				SortedSymbols:      []string{"Secret", "SecretPayload", "SecretStatus"},
				HasImports:         false,
			},
		},
		{
			name:       "file with external and internal imports",
			sourceFile: "google/cloud/asset/v1/assets.proto",
			messages: func() []*api.Message {
				extMsg := api.NewTestMessage("Duration").
					WithPackage("google.protobuf").
					WithSourceLocation("google/protobuf/duration.proto", 1)

				intMsg := api.NewTestMessage("ExportAssetsRequest").
					WithPackage("google.cloud.asset.v1").
					WithSourceLocation("google/cloud/asset/v1/asset_service.proto", 1)

				m := api.NewTestMessage("Asset").
					WithPackage("google.cloud.asset.v1").
					WithSourceLocation("google/cloud/asset/v1/assets.proto", 1).
					WithFields(
						api.NewTestField("duration").
							WithType(api.TypezMessage).
							WithMessageType(extMsg),
						api.NewTestField("export_req").
							WithType(api.TypezMessage).
							WithMessageType(intMsg),
					)
				return []*api.Message{m}
			}(),
			want: &fileAnnotations{
				SourceFile:         "google/cloud/asset/v1/assets.proto",
				Stem:               "assets",
				HasMessagesOrEnums: true,
				Manifest:           []string{"Asset"},
				SortedSymbols:      []string{"Asset"},
				HasImports:         true,
				ExternalImports: []externalImport{
					{
						Package: "google.protobuf.duration_pb2",
						Module:  "duration_pb2",
					},
				},
				InternalImports: []internalImport{
					{
						Package: "google.cloud.asset_v1.types",
						Module:  "asset_service",
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			allMessages := append([]*api.Message{}, test.messages...)
			model := api.NewTestAPI(allMessages, test.enums, nil)
			model.PackageName = "google.cloud.asset.v1"
			c := newTestCodec(t, model, nil)
			if err := c.annotateModel(); err != nil {
				t.Fatal(err)
			}

			ann := c.annotateFile(test.sourceFile, test.messages, test.enums)
			if diff := cmp.Diff(test.want, ann,
				cmpopts.IgnoreFields(fileAnnotations{},
					"Package", "CopyrightYear",
					"Messages", "Enums", "SortedMessages", "SortedEnums",
				),
			); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateFile_InternalImportAliasing(t *testing.T) {
	targetMsg := api.NewTestMessage("Asset").
		WithPackage("google.cloud.asset.v1").
		WithSourceLocation("google/cloud/asset/v1/assets.proto", 1)

	reqMsg := api.NewTestMessage("ExportAssetsRequest").
		WithPackage("google.cloud.asset.v1").
		WithSourceLocation("google/cloud/asset/v1/asset_service.proto", 1).
		WithFields(
			api.NewTestField("assets").
				WithType(api.TypezMessage).
				WithMessageType(targetMsg),
		)

	model := api.NewTestAPI([]*api.Message{targetMsg, reqMsg}, nil, nil)
	model.PackageName = "google.cloud.asset.v1"
	c := newTestCodec(t, model, nil)
	if err := c.annotateModel(); err != nil {
		t.Fatal(err)
	}

	ann := c.annotateFile("google/cloud/asset/v1/asset_service.proto", []*api.Message{reqMsg}, nil)
	want := []internalImport{
		{
			Package: "google.cloud.asset_v1.types",
			Module:  "assets",
			Alias:   "gca_assets",
		},
	}
	if diff := cmp.Diff(want, ann.InternalImports); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFileGrouping(t *testing.T) {
	msg1 := api.NewTestMessage("Msg1").
		WithSourceLocation("foo/bar/a.proto", 1)
	msg2 := api.NewTestMessage("Msg2").
		WithSourceLocation("foo/bar/b.proto", 1)
	enum1 := api.NewTestEnum("Enum1").
		WithSourceLocation("foo/bar/a.proto", 1)

	model := api.NewTestAPI([]*api.Message{msg1, msg2}, []*api.Enum{enum1}, nil)
	c := newTestCodec(t, model, nil)
	if err := c.annotateModel(); err != nil {
		t.Fatal(err)
	}

	ann, ok := model.Codec.(*modelAnnotations)
	if !ok {
		t.Fatalf("model.Codec = %T, want *modelAnnotations", model.Codec)
	}

	if got := len(ann.TypeFiles); got != 2 {
		t.Fatalf("annotateModel() TypeFiles len = %d, want 2", got)
	}

	t.Run("fileA", func(t *testing.T) {
		fileA := ann.TypeFiles["foo/bar/a.proto"]
		if fileA == nil {
			t.Fatal("missing TypeFile for foo/bar/a.proto")
		}
		var gotMsgNames, gotEnumNames []string
		for _, m := range fileA.Messages {
			gotMsgNames = append(gotMsgNames, m.Name)
		}
		for _, e := range fileA.Enums {
			gotEnumNames = append(gotEnumNames, e.Name)
		}
		t.Run("messages", func(t *testing.T) {
			if diff := cmp.Diff([]string{"Msg1"}, gotMsgNames); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
		t.Run("enums", func(t *testing.T) {
			if diff := cmp.Diff([]string{"Enum1"}, gotEnumNames); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	})

	t.Run("fileB", func(t *testing.T) {
		fileB := ann.TypeFiles["foo/bar/b.proto"]
		if fileB == nil {
			t.Fatal("missing TypeFile for foo/bar/b.proto")
		}
		var gotMsgNames []string
		for _, m := range fileB.Messages {
			gotMsgNames = append(gotMsgNames, m.Name)
		}
		t.Run("messages", func(t *testing.T) {
			if diff := cmp.Diff([]string{"Msg2"}, gotMsgNames); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	})
}
