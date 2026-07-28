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
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestProtobuf_FileOptions(t *testing.T) {
	requireProtoc(t)

	serviceConfig := &serviceconfig.Service{
		Name:  "testdata.googleapis.com",
		Title: "A test-only API",
	}
	got, err := makeAPIForProtobuf(serviceConfig, newTestCodeGeneratorRequest(t, "file_options.proto"))
	if err != nil {
		t.Fatal(err)
	}
	want := &api.API{
		Name:            "testdata",
		Title:           "A test-only API",
		CsharpNamespace: "Google.Cloud.TestData.V1",
		PhpNamespace:    "Google\\Cloud\\TestData\\V1",
		RubyPackage:     "Google::Cloud::TestData::V1",
	}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreUnexported(api.API{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestProtobuf_InconsistentFileOptions(t *testing.T) {
	requireProtoc(t)

	for _, test := range []struct {
		name      string
		filenames []string
	}{
		{
			name:      "C#",
			filenames: []string{"file_options.proto", "file_options_bad_csharp.proto"},
		},
		{
			name:      "PHP",
			filenames: []string{"file_options.proto", "file_options_bad_php.proto"},
		},
		{
			name:      "Ruby",
			filenames: []string{"file_options.proto", "file_options_bad_ruby.proto"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			serviceConfig := &serviceconfig.Service{
				Name:  "testdata.googleapis.com",
				Title: "A test-only API",
			}
			model, err := makeAPIForProtobuf(serviceConfig, newTestCodeGeneratorRequest(t, test.filenames...))
			if err == nil {
				t.Errorf("expected an error, got=%+v", model)
			}
		})
	}
}

func TestProtobuf_UpdateFileOption(t *testing.T) {
	for _, test := range []struct {
		name    string
		current string
		update  string
		want    string
	}{
		{"initial", "", "Value", "Value"},
		{"no update", "Value", "", "Value"},
		{"matches", "Value", "Value", "Value"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := protobufUpdateFileOption(test.current, test.update)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestProtobuf_UpdateFileOptionError(t *testing.T) {
	got, err := protobufUpdateFileOption("Value", "InconsistentValue")
	if err == nil {
		t.Errorf("expected an error, got=%s", got)
	}
}
