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

package rust

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestGenerateProstHybrid(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "cargo")
	msg := &api.Message{
		Name:    "Request",
		ID:      ".google.cloud.test.v1.Request",
		Package: "google.cloud.test.v1",
	}
	bidiService := &api.Service{
		Name:    "BidiService",
		ID:      ".google.cloud.test.v1.BidiService",
		Package: "google.cloud.test.v1",
		Methods: []*api.Method{
			{
				Name:                "Chat",
				ID:                  ".google.cloud.test.v1.BidiService.Chat",
				InputTypeID:         msg.ID,
				OutputTypeID:        msg.ID,
				InputType:           msg,
				OutputType:          msg,
				ClientSideStreaming: true,
				ServerSideStreaming: true,
				PathInfo:            &api.PathInfo{},
			},
		},
	}
	nonBidiService := &api.Service{
		Name:    "UnaryService",
		ID:      ".google.cloud.test.v1.UnaryService",
		Package: "google.cloud.test.v1",
		Methods: []*api.Method{
			{
				Name:         "Get",
				ID:           ".google.cloud.test.v1.UnaryService.Get",
				InputTypeID:  msg.ID,
				OutputTypeID: msg.ID,
				InputType:    msg,
				OutputType:   msg,
				PathInfo:     &api.PathInfo{},
			},
		},
	}

	bidiModel := api.NewTestAPI([]*api.Message{msg}, []*api.Enum{}, []*api.Service{bidiService})
	bidiModel.PackageName = "google.cloud.test.v1"
	if err := api.CrossReference(bidiModel); err != nil {
		t.Fatal(err)
	}

	nonBidiModel := api.NewTestAPI([]*api.Message{msg}, []*api.Enum{}, []*api.Service{nonBidiService})
	nonBidiModel.PackageName = "google.cloud.test.v1"
	if err := api.CrossReference(nonBidiModel); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name                        string
		model                       *api.API
		includeBidiStreamingMethods bool
		templateOverride            string
		wantProstDir                bool
	}{
		{
			name:                        "feature disabled does not create prost dir",
			model:                       bidiModel,
			includeBidiStreamingMethods: false,
			wantProstDir:                false,
		},
		{
			name:                        "model without bidi streaming does not create prost dir",
			model:                       nonBidiModel,
			includeBidiStreamingMethods: true,
			wantProstDir:                false,
		},
		{
			name:                        "template override tonic does not create prost dir",
			model:                       bidiModel,
			includeBidiStreamingMethods: true,
			templateOverride:            "tonic",
			wantProstDir:                false,
		},
		{
			name:                        "feature enabled creates prost dir",
			model:                       bidiModel,
			includeBidiStreamingMethods: true,
			wantProstDir:                true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()
			lib := &config.Library{
				Name: "test-package",
				Rust: &config.RustCrate{
					IncludeBidiStreamingMethods: test.includeBidiStreamingMethods,
					TemplateOverride:            test.templateOverride,
				},
			}
			absSpecSource, err := filepath.Abs("../../testdata/googleapis/google/type")
			if err != nil {
				t.Fatal(err)
			}
			srcs := &sources.Sources{
				Googleapis: filepath.Dir(filepath.Dir(absSpecSource)),
			}
			err = generateProstHybrid(t.Context(), test.model, lib, outDir, &parser.ModelConfig{
				SpecificationFormat: config.SpecProtobuf,
				SpecificationSource: absSpecSource,
				Source:              sources.NewSourceConfig(srcs, []string{"googleapis"}),
				Codec: map[string]string{
					"package-name-override": "google-cloud-test",
					"package:g3-wkt":        "package=google-cloud-wkt,source=google.protobuf",
				},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			prostDir := filepath.Join(outDir, "src", "prost")
			_, err = os.Stat(prostDir)
			exists := err == nil
			if exists != test.wantProstDir {
				t.Errorf("prostDir exists = %v, want %v", exists, test.wantProstDir)
			}

			convertFile := filepath.Join(outDir, "src", "convert.rs")
			_, err = os.Stat(convertFile)
			exists = err == nil
			if exists != test.wantProstDir {
				t.Errorf("convert file exists = %v, want %v", exists, test.wantProstDir)
			}
		})
	}
}

func TestFilterModelToStreaming(t *testing.T) {
	streamingMsg := &api.Message{
		Name:    "StreamMsg",
		ID:      ".google.test.v1.StreamMsg",
		Package: "google.test.v1",
	}
	unusedMsg := &api.Message{
		Name:    "UnusedMsg",
		ID:      ".google.test.v1.UnusedMsg",
		Package: "google.test.v1",
	}
	bidiService := &api.Service{
		Name:    "BidiService",
		ID:      ".google.test.v1.BidiService",
		Package: "google.test.v1",
		Methods: []*api.Method{
			{
				Name:                "Chat",
				ID:                  ".google.test.v1.BidiService.Chat",
				InputTypeID:         streamingMsg.ID,
				OutputTypeID:        streamingMsg.ID,
				InputType:           streamingMsg,
				OutputType:          streamingMsg,
				ClientSideStreaming: true,
				ServerSideStreaming: true,
			},
		},
	}
	model := api.NewTestAPI([]*api.Message{streamingMsg, unusedMsg}, []*api.Enum{}, []*api.Service{bidiService})
	model.PackageName = "google.test.v1"

	filtered, err := filterModelToStreaming(model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(filtered.Messages) != 1 || filtered.Messages[0].ID != streamingMsg.ID {
		t.Errorf("got messages %v, want [%s]", filtered.Messages, streamingMsg.ID)
	}

	if got := filtered.Message(unusedMsg.ID); got != unusedMsg {
		t.Errorf("filtered.Message(%q) = %v, want %v", unusedMsg.ID, got, unusedMsg)
	}
}

func TestFilterModelToStreamingNonStreamingFieldLookup(t *testing.T) {
	streamMsg := &api.Message{
		Name:    "StreamMsg",
		ID:      ".google.test.v1.StreamMsg",
		Package: "google.test.v1",
	}
	childData := &api.Message{
		Name:    "ChildData",
		ID:      ".google.test.v1.ChildData",
		Package: "google.test.v1",
	}
	unaryReq := &api.Message{
		Name:    "UnaryReq",
		ID:      ".google.test.v1.UnaryReq",
		Package: "google.test.v1",
		Fields: []*api.Field{
			{
				Name:    "info",
				TypezID: childData.ID,
				Typez:   api.TypezMessage,
			},
		},
	}
	bidiService := &api.Service{
		Name:    "BidiService",
		ID:      ".google.test.v1.BidiService",
		Package: "google.test.v1",
		Methods: []*api.Method{
			{
				Name:                "Chat",
				ID:                  ".google.test.v1.BidiService.Chat",
				InputTypeID:         streamMsg.ID,
				OutputTypeID:        streamMsg.ID,
				InputType:           streamMsg,
				OutputType:          streamMsg,
				ClientSideStreaming: true,
				ServerSideStreaming: true,
			},
		},
	}
	unaryService := &api.Service{
		Name:    "UnaryService",
		ID:      ".google.test.v1.UnaryService",
		Package: "google.test.v1",
		Methods: []*api.Method{
			{
				Name:                "UnaryMethod",
				ID:                  ".google.test.v1.UnaryService.UnaryMethod",
				InputTypeID:         unaryReq.ID,
				OutputTypeID:        unaryReq.ID,
				InputType:           unaryReq,
				OutputType:          unaryReq,
				ClientSideStreaming: false,
				ServerSideStreaming: false,
			},
		},
	}
	model := api.NewTestAPI([]*api.Message{streamMsg, unaryReq, childData}, []*api.Enum{}, []*api.Service{bidiService, unaryService})
	model.PackageName = "google.test.v1"

	filtered, err := filterModelToStreaming(model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(filtered.Messages) != 1 || filtered.Messages[0].ID != streamMsg.ID {
		t.Errorf("got messages %v, want [%s]", filtered.Messages, streamMsg.ID)
	}
	if got := filtered.Message(childData.ID); got != childData {
		t.Errorf("filtered.Message(%q) = %v, want %v", childData.ID, got, childData)
	}
	if got := filtered.Message(unaryReq.ID); got != unaryReq {
		t.Errorf("filtered.Message(%q) = %v, want %v", unaryReq.ID, got, unaryReq)
	}
}

func TestFilterModelToStreamingAnyError(t *testing.T) {
	// Verify google.protobuf.Any in streaming path returns error with recommendation
	anyMsg := &api.Message{
		Name:    "AnyReq",
		ID:      ".google.test.v1.AnyReq",
		Package: "google.test.v1",
		Fields: []*api.Field{
			{
				Name:    "details",
				TypezID: ".google.protobuf.Any",
				Typez:   api.TypezMessage,
			},
		},
	}
	anyService := &api.Service{
		Name:    "AnyService",
		ID:      ".google.test.v1.AnyService",
		Package: "google.test.v1",
		Methods: []*api.Method{
			{
				Name:                "ChatAny",
				ID:                  ".google.test.v1.AnyService.ChatAny",
				InputTypeID:         anyMsg.ID,
				OutputTypeID:        anyMsg.ID,
				InputType:           anyMsg,
				OutputType:          anyMsg,
				ClientSideStreaming: true,
				ServerSideStreaming: true,
			},
		},
	}
	anyModel := api.NewTestAPI([]*api.Message{anyMsg}, []*api.Enum{}, []*api.Service{anyService})
	anyModel.PackageName = "google.test.v1"

	_, err := filterModelToStreaming(anyModel)
	if err == nil {
		t.Fatal("expected error for google.protobuf.Any, got nil")
	}
	if !strings.Contains(err.Error(), "skipped_ids") {
		t.Errorf("expected error to contain recommendation 'skipped_ids', got: %v", err)
	}
}
