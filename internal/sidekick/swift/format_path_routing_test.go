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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestResolveRoutingFieldPath(t *testing.T) {
	makeFixture := func(specFormat string) (*codec, *api.Message) {
		statusEnum := api.NewTestEnum("Status").
			WithPackage("google.cloud.test.v1").
			WithValues(
				api.NewTestEnumValue("STATUS_UNSPECIFIED", 0),
				api.NewTestEnumValue("STATUS_ACTIVE", 1),
			)
		extEnum := api.NewTestEnum("ExtMode").
			WithPackage("google.type").
			WithValues(
				api.NewTestEnumValue("EXT_MODE_UNSPECIFIED", 0),
				api.NewTestEnumValue("EXT_MODE_ON", 1),
			)
		unconfiguredEnum := api.NewTestEnum("UnconfiguredEnum").
			WithPackage("google.unconfigured").
			WithValues(
				api.NewTestEnumValue("UNCONFIGURED_UNSPECIFIED", 0),
			)

		extContainerMsg := api.NewTestMessage("ExtContainer").
			WithPackage("google.type").
			WithFields(
				api.NewTestField("rank").WithType(api.TypezInt32),
				api.NewTestField("unconfigured_sibling").WithEnumType(unconfiguredEnum),
			)

		innerMsg := api.NewTestMessage("Inner").
			WithPackage("google.cloud.test.v1").
			WithFields(
				api.NewTestField("count").WithType(api.TypezInt32),
				api.NewTestField("opt_count").WithType(api.TypezInt32).WithOptional(),
				api.NewTestField("status").WithEnumType(statusEnum),
				api.NewTestField("opt_status").WithEnumType(statusEnum).WithOptional(),
			)

		requestMsg := api.NewTestMessage("Request").
			WithPackage("google.cloud.test.v1").
			WithFields(
				api.NewTestField("name").WithType(api.TypezString),
				api.NewTestField("opt_name").WithType(api.TypezString).WithOptional(),
				api.NewTestField("page_size").WithType(api.TypezInt32),
				api.NewTestField("opt_page_size").WithType(api.TypezInt32).WithOptional(),
				api.NewTestField("ratio").WithType(api.TypezDouble),
				api.NewTestField("enabled").WithType(api.TypezBool),
				api.NewTestField("status").WithEnumType(statusEnum),
				api.NewTestField("opt_status").WithEnumType(statusEnum).WithOptional(),
				api.NewTestField("ext_mode").WithEnumType(extEnum),
				api.NewTestField("ext_container").WithMessageType(extContainerMsg).WithOptional(),
				api.NewTestField("req_inner").WithMessageType(innerMsg),
				api.NewTestField("info").WithMessageType(innerMsg).WithOptional(),
			)

		model := api.NewTestAPI(
			[]*api.Message{innerMsg, requestMsg},
			[]*api.Enum{statusEnum, extEnum},
			nil,
		).WithPackageName("google.cloud.test.v1")
		model.AddMessage(extContainerMsg)
		model.AddEnum(unconfiguredEnum)
		if err := api.CrossReference(model); err != nil {
			t.Fatal(err)
		}
		lib := &config.Library{
			SpecificationFormat: specFormat,
			Swift: swiftConfig(t, []config.SwiftDependency{
				{Name: "GoogleType", ApiPackage: "google.type"},
			}),
		}
		c := newTestCodec(t, model, lib)
		if err := c.annotateModel(); err != nil {
			t.Fatal(err)
		}
		return c, requestMsg
	}

	for _, test := range []struct {
		name       string
		specFormat string
		fieldPath  []string
		want       *routingParamVariant
	}{
		{
			name:       "empty field path",
			specFormat: config.SpecProtobuf,
			fieldPath:  nil,
			want:       nil,
		},
		{
			name:       "non-optional string",
			specFormat: config.SpecProtobuf,
			fieldPath:  []string{"name"},
			want: &routingParamVariant{
				FieldSteps: []*routingFieldStep{{PropertyName: "name"}},
			},
		},
		{
			name:       "optional string",
			specFormat: config.SpecProtobuf,
			fieldPath:  []string{"opt_name"},
			want: &routingParamVariant{
				FieldSteps: []*routingFieldStep{{PropertyName: "optName"}},
				IsOptional: true,
			},
		},
		{
			name:       "non-optional int32 scalar",
			specFormat: config.SpecProtobuf,
			fieldPath:  []string{"page_size"},
			want: &routingParamVariant{
				FieldSteps:      []*routingFieldStep{{PropertyName: "pageSize"}},
				ConvertToString: true,
			},
		},
		{
			name:       "optional int32 scalar",
			specFormat: config.SpecProtobuf,
			fieldPath:  []string{"opt_page_size"},
			want: &routingParamVariant{
				FieldSteps:      []*routingFieldStep{{PropertyName: "optPageSize"}},
				ConvertToString: true,
				IsOptional:      true,
			},
		},
		{
			name:       "double scalar",
			specFormat: config.SpecProtobuf,
			fieldPath:  []string{"ratio"},
			want: &routingParamVariant{
				FieldSteps:      []*routingFieldStep{{PropertyName: "ratio"}},
				ConvertToString: true,
			},
		},
		{
			name:       "bool scalar",
			specFormat: config.SpecProtobuf,
			fieldPath:  []string{"enabled"},
			want: &routingParamVariant{
				FieldSteps:      []*routingFieldStep{{PropertyName: "enabled"}},
				ConvertToString: true,
			},
		},
		{
			name:       "non-optional protobuf enum",
			specFormat: config.SpecProtobuf,
			fieldPath:  []string{"status"},
			want: &routingParamVariant{
				FieldSteps: []*routingFieldStep{{PropertyName: "status"}},
				IsEnum:     true,
			},
		},
		{
			name:       "optional protobuf enum",
			specFormat: config.SpecProtobuf,
			fieldPath:  []string{"opt_status"},
			want: &routingParamVariant{
				FieldSteps: []*routingFieldStep{{PropertyName: "optStatus"}},
				IsEnum:     true,
				IsOptional: true,
			},
		},
		{
			name:       "external enum records required package",
			specFormat: config.SpecProtobuf,
			fieldPath:  []string{"ext_mode"},
			want: &routingParamVariant{
				FieldSteps:       []*routingFieldStep{{PropertyName: "extMode"}},
				IsEnum:           true,
				RequiredPackages: []string{"google.type"},
			},
		},
		{
			name:       "external unannotated intermediate ignores unconfigured sibling",
			specFormat: config.SpecProtobuf,
			fieldPath:  []string{"ext_container", "rank"},
			want: &routingParamVariant{
				FieldSteps: []*routingFieldStep{
					{PropertyName: "extContainer"},
					{PropertyName: "rank", ChainOptional: true},
				},
				ConvertToString:  true,
				IsOptional:       true,
				NeedsParens:      true,
				RequiredPackages: []string{"google.type"},
			},
		},
		{
			name:       "discovery string enum non-optional",
			specFormat: config.SpecDiscovery,
			fieldPath:  []string{"status"},
			want: &routingParamVariant{
				FieldSteps:   []*routingFieldStep{{PropertyName: "status"}},
				IsEnum:       true,
				IsStringEnum: true,
			},
		},
		{
			name:       "discovery string enum optional",
			specFormat: config.SpecDiscovery,
			fieldPath:  []string{"opt_status"},
			want: &routingParamVariant{
				FieldSteps:   []*routingFieldStep{{PropertyName: "optStatus"}},
				IsEnum:       true,
				IsStringEnum: true,
				IsOptional:   true,
			},
		},
		{
			name:       "non-optional message to optional scalar does not need parens",
			specFormat: config.SpecProtobuf,
			fieldPath:  []string{"req_inner", "opt_count"},
			want: &routingParamVariant{
				FieldSteps: []*routingFieldStep{
					{PropertyName: "reqInner"},
					{PropertyName: "optCount"},
				},
				ConvertToString: true,
				IsOptional:      true,
				NeedsParens:     false,
			},
		},
		{
			name:       "nested optional message to scalar",
			specFormat: config.SpecProtobuf,
			fieldPath:  []string{"info", "count"},
			want: &routingParamVariant{
				FieldSteps: []*routingFieldStep{
					{PropertyName: "info"},
					{PropertyName: "count", ChainOptional: true},
				},
				ConvertToString: true,
				IsOptional:      true,
				NeedsParens:     true,
			},
		},
		{
			name:       "nested optional message to enum",
			specFormat: config.SpecProtobuf,
			fieldPath:  []string{"info", "opt_status"},
			want: &routingParamVariant{
				FieldSteps: []*routingFieldStep{
					{PropertyName: "info"},
					{PropertyName: "optStatus", ChainOptional: true},
				},
				IsEnum:      true,
				IsOptional:  true,
				NeedsParens: true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, requestMsg := makeFixture(test.specFormat)
			got, err := c.resolveRoutingFieldPath(requestMsg, test.fieldPath)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolveRoutingFieldPath_Error(t *testing.T) {
	innerMsg := api.NewTestMessage("Inner").
		WithPackage("google.cloud.test.v1").
		WithFields(
			api.NewTestField("name").WithType(api.TypezString),
		)
	nodeMsg := api.NewTestMessage("Node").
		WithPackage("google.cloud.test.v1").
		WithFields(
			api.NewTestField("rank").WithType(api.TypezInt32),
		)
	nodeMsg.WithFields(
		api.NewTestField("child").WithMessageType(nodeMsg).WithOptional(),
	)
	stringMap := api.NewTestMap("LabelsEntry", api.TypezString, api.TypezString)
	requestMsg := api.NewTestMessage("Request").
		WithPackage("google.cloud.test.v1").
		WithFields(
			api.NewTestField("name").WithType(api.TypezString),
			api.NewTestField("tags").WithType(api.TypezString).WithRepeated(),
			api.NewTestField("labels").WithMapType(stringMap).WithMap(),
			api.NewTestField("payload").WithType(api.TypezBytes),
			api.NewTestField("bytes_wrapper").WithType(api.TypezMessage).WithTypezID(api.WktBytesValueID).WithOptional(),
			api.NewTestField("int64_wrapper").WithType(api.TypezMessage).WithTypezID(api.WktInt64ValueID).WithOptional(),
			api.NewTestField("null_value").WithType(api.TypezEnum).WithTypezID(api.WktNullValueID).WithOptional(),
			api.NewTestField("node").WithMessageType(nodeMsg).WithOptional().WithRecursive(),
			api.NewTestField("inner").WithMessageType(innerMsg).WithOptional(),
		).
		WithOneOfs(
			api.NewTestOneOf("source").WithFields(
				api.NewTestField("bucket").WithType(api.TypezString),
			),
		)
	model := api.NewTestAPI([]*api.Message{innerMsg, nodeMsg, requestMsg}, nil, nil).
		WithPackageName("google.cloud.test.v1").
		WithMaps(stringMap)
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}
	c := newTestCodec(t, model, &config.Library{Swift: swiftConfig(t, nil)})
	if err := c.annotateModel(); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name      string
		message   *api.Message
		fieldPath []string
		wantErr   string
	}{
		{
			name:      "nil message",
			message:   nil,
			fieldPath: []string{"name"},
			wantErr:   "missing input message",
		},
		{
			name:      "missing field",
			message:   requestMsg,
			fieldPath: []string{"does_not_exist"},
			wantErr:   "does_not_exist",
		},
		{
			name:      "repeated field",
			message:   requestMsg,
			fieldPath: []string{"tags"},
			wantErr:   "tags",
		},
		{
			name:      "map field",
			message:   requestMsg,
			fieldPath: []string{"labels"},
			wantErr:   "labels",
		},
		{
			name:      "bytes field",
			message:   requestMsg,
			fieldPath: []string{"payload"},
			wantErr:   "payload",
		},
		{
			name:      "WKT BytesValue field",
			message:   requestMsg,
			fieldPath: []string{"bytes_wrapper"},
			wantErr:   api.WktBytesValueID,
		},
		{
			name:      "WKT Int64Value wrapper field",
			message:   requestMsg,
			fieldPath: []string{"int64_wrapper"},
			wantErr:   api.WktInt64ValueID,
		},
		{
			name:      "WKT NullValue field",
			message:   requestMsg,
			fieldPath: []string{"null_value"},
			wantErr:   api.WktNullValueID,
		},
		{
			name:      "oneof field",
			message:   requestMsg,
			fieldPath: []string{"bucket"},
			wantErr:   "bucket",
		},
		{
			name:      "recursive field",
			message:   requestMsg,
			fieldPath: []string{"node", "rank"},
			wantErr:   "node",
		},
		{
			name:      "non-message intermediate field",
			message:   requestMsg,
			fieldPath: []string{"name", "sub"},
			wantErr:   "not a message",
		},
		{
			name:      "non-wrapper message leaf",
			message:   requestMsg,
			fieldPath: []string{"inner"},
			wantErr:   "inner",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, gotErr := c.resolveRoutingFieldPath(test.message, test.fieldPath)
			if gotErr == nil {
				t.Fatalf("resolveRoutingFieldPath(%v) succeeded with %+v, want error", test.fieldPath, got)
			}
			if !strings.Contains(gotErr.Error(), test.wantErr) {
				t.Errorf("resolveRoutingFieldPath(%v) error = %q, want substring %q", test.fieldPath, gotErr.Error(), test.wantErr)
			}
		})
	}
}
