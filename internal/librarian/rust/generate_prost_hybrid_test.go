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
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestGenerateProstHybrid(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "cargo")
	msg := api.NewTestMessage("Request").WithPackage("google.cloud.test.v1")

	chatMethod := api.NewTestMethod("Chat").WithInput(msg).WithOutput(msg).WithBidiStreaming()

	bidiService := api.NewTestService("BidiService").WithPackage("google.cloud.test.v1").WithMethods(chatMethod)

	getMethod := api.NewTestMethod("Get").WithInput(msg).WithOutput(msg)
	nonBidiService := api.NewTestService("UnaryService").WithPackage("google.cloud.test.v1").WithMethods(getMethod)

	bidiModel := api.NewTestAPI([]*api.Message{msg}, []*api.Enum{}, []*api.Service{bidiService})
	if err := api.CrossReference(bidiModel); err != nil {
		t.Fatal(err)
	}

	nonBidiModel := api.NewTestAPI([]*api.Message{msg}, []*api.Enum{}, []*api.Service{nonBidiService})
	if err := api.CrossReference(nonBidiModel); err != nil {
		t.Fatal(err)
	}

	expandMethod := api.NewTestMethod("Expand").WithInput(msg).WithOutput(msg).WithServerSideStreaming()
	serverService := api.NewTestService("ServerService").WithPackage("google.cloud.test.v1").WithMethods(expandMethod)
	serverStreamingModel := api.NewTestAPI([]*api.Message{msg}, []*api.Enum{}, []*api.Service{serverService})
	if err := api.CrossReference(serverStreamingModel); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name                          string
		model                         *api.API
		includeBidiStreamingMethods   bool
		includeServerStreamingMethods bool
		templateOverride              string
		wantProstDir                  bool
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
		{
			name:                          "server streaming enabled creates prost dir",
			model:                         serverStreamingModel,
			includeServerStreamingMethods: true,
			wantProstDir:                  true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()
			lib := &config.Library{
				Name: "test-package",
				Rust: &config.RustCrate{
					RustDefault: config.RustDefault{
						IncludeBidiStreamingMethods:   &test.includeBidiStreamingMethods,
						IncludeServerStreamingMethods: &test.includeServerStreamingMethods,
					},
					TemplateOverride: test.templateOverride,
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
	streamingMsg := api.NewTestMessage("StreamMsg").WithPackage("google.test.v1")
	unusedMsg := api.NewTestMessage("UnusedMsg").WithPackage("google.test.v1")

	chatMethod := api.NewTestMethod("Chat").WithInput(streamingMsg).WithOutput(streamingMsg).WithBidiStreaming()

	bidiService := api.NewTestService("BidiService").WithPackage("google.test.v1").WithMethods(chatMethod)
	model := api.NewTestAPI([]*api.Message{streamingMsg, unusedMsg}, []*api.Enum{}, []*api.Service{bidiService})

	filtered, unused, _, err := filterModelToStreaming(model, true, true, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(filtered.Messages) != 1 || filtered.Messages[0].ID != streamingMsg.ID {
		t.Errorf("got messages %v, want [%s]", filtered.Messages, streamingMsg.ID)
	}
	if len(unused) != 1 || unused[0] != unusedMsg.ID {
		t.Errorf("got unused %v, want [%s]", unused, unusedMsg.ID)
	}

	if got := filtered.Message(unusedMsg.ID); got != unusedMsg {
		t.Errorf("filtered.Message(%q) = %v, want %v", unusedMsg.ID, got, unusedMsg)
	}
}

func TestFilterModelToStreamingNonStreamingFieldLookup(t *testing.T) {
	streamMsg := api.NewTestMessage("StreamMsg").WithPackage("google.test.v1")
	childData := api.NewTestMessage("ChildData").WithPackage("google.test.v1")
	unaryReq := api.NewTestMessage("UnaryReq").WithPackage("google.test.v1").WithFields(
		&api.Field{
			Name:    "info",
			TypezID: childData.ID,
			Typez:   api.TypezMessage,
		},
	)

	chatMethod := api.NewTestMethod("Chat").WithInput(streamMsg).WithOutput(streamMsg).WithBidiStreaming()
	bidiService := api.NewTestService("BidiService").WithPackage("google.test.v1").WithMethods(chatMethod)

	unaryMethod := api.NewTestMethod("UnaryMethod").WithInput(unaryReq).WithOutput(unaryReq)
	unaryService := api.NewTestService("UnaryService").WithPackage("google.test.v1").WithMethods(unaryMethod)

	model := api.NewTestAPI([]*api.Message{streamMsg, unaryReq, childData}, []*api.Enum{}, []*api.Service{bidiService, unaryService})

	filtered, _, _, err := filterModelToStreaming(model, true, true, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(filtered.Services) != 1 || filtered.Services[0].ID != bidiService.ID {
		t.Errorf("got services %v, want [%s]", filtered.Services, bidiService.ID)
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

func TestFilterModelToStreamingExternalTypes(t *testing.T) {
	streamMsg := api.NewTestMessage("StreamMsg").WithPackage("google.test.v1")
	externalMsg := api.NewTestMessage("LatLng").WithPackage("google.type")
	externalEnum := &api.Enum{Name: "DayOfWeek", ID: ".google.type.DayOfWeek", Package: "google.type"}

	streamMsg.Fields = []*api.Field{
		{
			Name:    "location",
			TypezID: externalMsg.ID,
			Typez:   api.TypezMessage,
		},
		{
			Name:    "day",
			TypezID: externalEnum.ID,
			Typez:   api.TypezEnum,
		},
	}

	chatMethod := api.NewTestMethod("Chat").WithInput(streamMsg).WithOutput(streamMsg).WithBidiStreaming()
	bidiService := api.NewTestService("BidiService").WithPackage("google.test.v1").WithMethods(chatMethod)

	model := api.NewTestAPI([]*api.Message{streamMsg}, []*api.Enum{}, []*api.Service{bidiService})
	model.AddMessage(externalMsg)
	model.AddEnum(externalEnum)
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	filtered, _, _, err := filterModelToStreaming(model, true, true, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(filtered.Messages) != 1 || filtered.Messages[0].ID != streamMsg.ID {
		t.Errorf("got package messages %v, want [%s]", filtered.Messages, streamMsg.ID)
	}
	if len(filtered.ExternalMessages) != 1 || filtered.ExternalMessages[0].ID != externalMsg.ID {
		t.Errorf("got ExternalMessages %v, want [%s]", filtered.ExternalMessages, externalMsg.ID)
	}
	if len(filtered.ExternalEnums) != 1 || filtered.ExternalEnums[0].ID != externalEnum.ID {
		t.Errorf("got ExternalEnums %v, want [%s]", filtered.ExternalEnums, externalEnum.ID)
	}
}

func TestFilterModelToStreamingAnyError(t *testing.T) {
	// Verify google.protobuf.Any in streaming path returns error with recommendation
	anyMsg := api.NewTestMessage("AnyReq").WithPackage("google.test.v1").WithFields(
		&api.Field{
			Name:    "details",
			TypezID: ".google.protobuf.Any",
			Typez:   api.TypezMessage,
		},
	)
	chatAnyMethod := api.NewTestMethod("ChatAny").WithInput(anyMsg).WithOutput(anyMsg).WithBidiStreaming()
	anyService := api.NewTestService("AnyService").WithPackage("google.test.v1").WithMethods(chatAnyMethod)

	anyModel := api.NewTestAPI([]*api.Message{anyMsg}, []*api.Enum{}, []*api.Service{anyService})

	_, _, _, err := filterModelToStreaming(anyModel, true, true, nil)
	if err == nil {
		t.Fatal("expected error for google.protobuf.Any, got nil")
	}
	if !strings.Contains(err.Error(), "allow_streaming_any_types") {
		t.Errorf("expected error to contain recommendation 'allow_streaming_any_types', got: %v", err)
	}
}

func TestFilterModelToStreamingAllowedAny(t *testing.T) {
	// Verify allowing specific Any field path succeeds and drops the field from conversion
	anyMsg := api.NewTestMessage("AnyReq").WithPackage("google.test.v1").WithFields(
		api.NewTestField("details").WithType(api.TypezMessage).WithTypezID(".google.protobuf.Any"),
		api.NewTestField("name").WithType(api.TypezString),
	)
	chatAnyMethod := api.NewTestMethod("ChatAny").WithInput(anyMsg).WithOutput(anyMsg).WithBidiStreaming()
	anyService := api.NewTestService("AnyService").WithPackage("google.test.v1").WithMethods(chatAnyMethod)

	anyModel := api.NewTestAPI([]*api.Message{anyMsg}, []*api.Enum{}, []*api.Service{anyService})

	filtered, _, _, err := filterModelToStreaming(anyModel, true, true, []string{".google.test.v1.AnyReq.details"})
	if err != nil {
		t.Fatalf("expected success with allowStreamingAnyTypes, got error: %v", err)
	}

	msg := filtered.Message(anyMsg.ID)
	if msg == nil {
		t.Fatalf("expected message %s in filtered model", anyMsg.ID)
	}
	if !msg.HasSkippedProtoConversionFields() {
		t.Errorf("expected msg.HasSkippedProtoConversionFields() to be true")
	}
	if len(msg.Fields) != 2 {
		t.Errorf("expected msg.Fields to retain all 2 fields, got: %v", msg.Fields)
	}
	detailsField := slices.IndexFunc(msg.Fields, func(f *api.Field) bool { return f.Name == "details" })
	if detailsField == -1 || !msg.Fields[detailsField].SkipProtoConversion {
		t.Errorf("expected 'details' field to have SkipProtoConversion == true")
	}
	nameField := slices.IndexFunc(msg.Fields, func(f *api.Field) bool { return f.Name == "name" })
	if nameField == -1 || msg.Fields[nameField].SkipProtoConversion {
		t.Errorf("expected 'name' field to have SkipProtoConversion == false")
	}
}

func TestFilterModelToStreamingGoogleRpcStatus(t *testing.T) {
	// Verify google.rpc.Status in streaming path succeeds and does not error on Any details
	statusMsg := api.NewTestMessage("Status").WithPackage("google.rpc").WithFields(
		&api.Field{
			Name:  "code",
			Typez: api.TypezInt32,
		},
		&api.Field{
			Name:  "message",
			Typez: api.TypezString,
		},
		&api.Field{
			Name:     "details",
			TypezID:  ".google.protobuf.Any",
			Typez:    api.TypezMessage,
			Repeated: true,
		},
	)

	reqMsg := api.NewTestMessage("StreamReq").WithPackage("google.test.v1").WithFields(
		&api.Field{
			Name:    "status",
			TypezID: statusMsg.ID,
			Typez:   api.TypezMessage,
		},
	)

	chatMethod := api.NewTestMethod("ChatStatus").WithInput(reqMsg).WithOutput(reqMsg).WithBidiStreaming()
	statusService := api.NewTestService("StatusService").WithPackage("google.test.v1").WithMethods(chatMethod)

	statusModel := api.NewTestAPI([]*api.Message{reqMsg}, []*api.Enum{}, []*api.Service{statusService})
	statusModel.AddMessage(statusMsg)

	filtered, unused, hasStatus, err := filterModelToStreaming(statusModel, true, true, nil)
	if err != nil {
		t.Fatalf("unexpected error for google.rpc.Status: %v", err)
	}

	if len(filtered.Messages) != 1 {
		t.Errorf("got %d messages, want 1 (StreamReq only, google.rpc.Status handled via Codec)", len(filtered.Messages))
	}

	if !hasStatus {
		t.Errorf("expected hasStatus boolean from filterModelToStreaming to be true, got false")
	}

	for _, u := range unused {
		if u == statusMsg.ID {
			t.Errorf("google.rpc.Status should not be in unused list, got: %v", unused)
		}
	}
}

func TestFilterModelToStreamingNestedTypeParentPreservation(t *testing.T) {
	parent := api.NewTestMessage("Parent").WithPackage("google.test.v1")
	child := api.NewTestMessage("Child").WithPackage("google.test.v1")
	sibling := api.NewTestMessage("Sibling").WithPackage("google.test.v1")
	child.Parent = parent

	parent.Fields = []*api.Field{
		{
			Name:    "sibling_ref",
			TypezID: sibling.ID,
			Typez:   api.TypezMessage,
		},
	}

	streamReq := api.NewTestMessage("StreamReq").WithPackage("google.test.v1").WithFields(
		&api.Field{
			Name:    "child_ref",
			TypezID: child.ID,
			Typez:   api.TypezMessage,
		},
	)

	chatMethod := api.NewTestMethod("Chat").WithInput(streamReq).WithOutput(streamReq).WithBidiStreaming()
	bidiService := api.NewTestService("BidiService").WithPackage("google.test.v1").WithMethods(chatMethod)

	model := api.NewTestAPI([]*api.Message{parent, child, sibling, streamReq}, []*api.Enum{}, []*api.Service{bidiService})

	_, unusedTypes, _, err := filterModelToStreaming(model, true, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, unused := range unusedTypes {
		if unused == parent.ID {
			t.Errorf("parent message %q should not be in unusedTypes", parent.ID)
		}
		if unused == child.ID {
			t.Errorf("child message %q should not be in unusedTypes", child.ID)
		}
		if unused == sibling.ID {
			t.Errorf("sibling field message %q of parent should not be in unusedTypes", sibling.ID)
		}
	}
}

func TestFilterModelToStreamingServerStreaming(t *testing.T) {
	reqMsg := api.NewTestMessage("ExpandRequest").WithPackage("google.test.v1")
	respMsg := api.NewTestMessage("EchoResponse").WithPackage("google.test.v1")
	unusedMsg := api.NewTestMessage("UnusedMsg").WithPackage("google.test.v1")

	expandMethod := api.NewTestMethod("Expand").WithInput(reqMsg).WithOutput(respMsg).WithServerSideStreaming()

	serverService := api.NewTestService("EchoService").WithPackage("google.test.v1").WithMethods(expandMethod)
	model := api.NewTestAPI([]*api.Message{reqMsg, respMsg, unusedMsg}, []*api.Enum{}, []*api.Service{serverService})

	filtered, unused, _, err := filterModelToStreaming(model, false, true, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(filtered.Services) != 1 || filtered.Services[0].ID != serverService.ID {
		t.Errorf("got services %v, want [%s]", filtered.Services, serverService.ID)
	}
	if len(filtered.Messages) != 2 {
		t.Errorf("got messages %v, want [%s, %s]", filtered.Messages, reqMsg.ID, respMsg.ID)
	}
	if len(unused) != 1 || unused[0] != unusedMsg.ID {
		t.Errorf("got unused %v, want [%s]", unused, unusedMsg.ID)
	}
}

func TestFilterModelToStreamingIsolation(t *testing.T) {
	bidiMsg := api.NewTestMessage("BidiMsg").WithPackage("google.test.v1")
	serverReq := api.NewTestMessage("ServerReq").WithPackage("google.test.v1")
	serverResp := api.NewTestMessage("ServerResp").WithPackage("google.test.v1")

	bidiMethod := api.NewTestMethod("Chat").WithInput(bidiMsg).WithOutput(bidiMsg).WithBidiStreaming()
	serverMethod := api.NewTestMethod("Expand").WithInput(serverReq).WithOutput(serverResp).WithServerSideStreaming()

	service := api.NewTestService("MixedService").WithPackage("google.test.v1").WithMethods(bidiMethod, serverMethod)
	model := api.NewTestAPI([]*api.Message{bidiMsg, serverReq, serverResp}, []*api.Enum{}, []*api.Service{service})

	for _, test := range []struct {
		name          string
		includeBidi   bool
		includeServer bool
		wantMessages  []string
		wantUnused    []string
	}{
		{
			name:          "server streaming only ignores bidi types",
			includeBidi:   false,
			includeServer: true,
			wantMessages:  []string{serverReq.ID, serverResp.ID},
			wantUnused:    []string{bidiMsg.ID},
		},
		{
			name:          "bidi streaming only ignores server streaming types",
			includeBidi:   true,
			includeServer: false,
			wantMessages:  []string{bidiMsg.ID},
			wantUnused:    []string{serverReq.ID, serverResp.ID},
		},
		{
			name:          "both streaming types enabled includes all types",
			includeBidi:   true,
			includeServer: true,
			wantMessages:  []string{bidiMsg.ID, serverReq.ID, serverResp.ID},
			wantUnused:    nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			filtered, unused, _, err := filterModelToStreaming(model, test.includeBidi, test.includeServer, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			var gotMessages []string
			for _, m := range filtered.Messages {
				gotMessages = append(gotMessages, m.ID)
			}
			less := func(a, b string) bool { return a < b }
			if diff := cmp.Diff(test.wantMessages, gotMessages, cmpopts.SortSlices(less), cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("mismatch in filtered messages (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantUnused, unused, cmpopts.SortSlices(less), cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("mismatch in unused messages (-want +got):\n%s", diff)
			}
		})
	}
}
