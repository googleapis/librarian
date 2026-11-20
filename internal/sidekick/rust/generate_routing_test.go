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

	"github.com/google/go-cmp/cmp"
	libconfig "github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

func TestGenerateExplicitEmptyRouting(t *testing.T) {
	for _, test := range []struct {
		name     string
		filePath string
		codec    map[string]string
	}{
		{
			name:     "crate template",
			filePath: filepath.Join("src", "transport.rs"),
			codec: map[string]string{
				"package:wkt":       "source=google.protobuf,package=google-cloud-wkt",
				"default-transport": "grpc",
				"routing-required":  "true",
			},
		},
		{
			name:     "grpc-client template",
			filePath: "transport.rs",
			codec: map[string]string{
				"template-override": "templates/grpc-client",
				"package:wkt":       "source=google.protobuf,package=google-cloud-wkt",
				"routing-required":  "true",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := api.NewTestMessage("Request").
				WithPackage("test.v1").
				WithFields(
					api.NewTestField("table_name").WithType(api.TypezString),
				)
			response := api.NewTestMessage("Response").WithPackage("test.v1")

			explicitEmptyUnary := api.NewTestMethod("ExplicitEmpty").
				WithInput(request).
				WithOutput(response).
				WithVerb("GET").
				WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").WithVariableNamed("table_name").Build()).
				WithRouting()

			explicitEmptyStreaming := api.NewTestMethod("ExplicitEmptyStream").
				WithInput(request).
				WithOutput(response).
				WithServerSideStreaming().
				WithVerb("GET").
				WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").WithVariableNamed("table_name").Build()).
				WithRouting()

			service := api.NewTestService("TestService").
				WithPackage("test.v1").
				WithMethods(explicitEmptyUnary, explicitEmptyStreaming)

			model := api.NewTestAPI([]*api.Message{request, response}, nil, []*api.Service{service}).
				WithPackageName("test.v1")
			if err := api.CrossReference(model); err != nil {
				t.Fatal(err)
			}

			outDir := t.TempDir()
			cfg := &parser.ModelConfig{
				SpecificationFormat: libconfig.SpecProtobuf,
				Codec:               test.codec,
			}
			if err := Generate(t.Context(), model, outDir, cfg); err != nil {
				t.Fatal(err)
			}

			contentBytes, err := os.ReadFile(filepath.Join(outDir, test.filePath))
			if err != nil {
				t.Fatal(err)
			}
			content := string(contentBytes)

			if got := strings.Count(content, "let x_goog_request_params = String::new();"); got != 2 {
				t.Errorf("expected 2 empty routing header bindings, got %d in:\n%s", got, content)
			}
			for _, forbidden := range []string{
				"table_name={v}",
				"BindingError",
				"use gaxi::routing_parameter::Segment;",
			} {
				if strings.Contains(content, forbidden) {
					t.Errorf("expected generated %s not to contain %q in:\n%s", test.filePath, forbidden, content)
				}
			}
		})
	}
}

type implicitHTTPRoutingCase struct {
	name       string
	filePath   string
	codec      map[string]string
	endMarkers map[string]string
}

func TestGenerateImplicitHTTPRouting(t *testing.T) {
	for _, test := range []implicitHTTPRoutingCase{
		{
			name:     "crate template",
			filePath: filepath.Join("src", "transport.rs"),
			codec: map[string]string{
				"package:wkt":               "source=google.protobuf,package=google-cloud-wkt",
				"default-transport":         "grpc",
				"include-grpc-only-methods": "true",
			},
			endMarkers: map[string]string{
				"unary":  "\n        type TR =",
				"stream": "\n        let extensions =",
			},
		},
		{
			name:     "grpc-client template",
			filePath: "transport.rs",
			codec: map[string]string{
				"template-override":         "templates/grpc-client",
				"package:wkt":               "source=google.protobuf,package=google-cloud-wkt",
				"include-grpc-only-methods": "true",
			},
			endMarkers: map[string]string{
				"unary":  "\n        type TR =",
				"stream": "\n        self.inner",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			verifyImplicitHTTPRouting(t, test)
		})
	}
}

func verifyImplicitHTTPRouting(t *testing.T, tc implicitHTTPRoutingCase) {
	t.Helper()

	kindEnum := api.NewTestEnum("Kind").
		WithPackage("test.v1").
		WithValues(api.NewTestEnumValue("KIND_UNSPECIFIED", 0))
	child := api.NewTestMessage("Child").
		WithPackage("test.v1").
		WithFields(
			api.NewTestField("id").WithType(api.TypezString),
		)
	oneofStrField := api.NewTestField("oneofStr").WithType(api.TypezString)
	oneofStr := api.NewTestOneOf("strChoice").WithFields(oneofStrField)
	oneofNumField := api.NewTestField("oneofNum").WithType(api.TypezInt32)
	oneofNum := api.NewTestOneOf("numChoice").WithFields(oneofNumField)

	request := api.NewTestMessage("Request").
		WithPackage("test.v1").
		WithFields(
			api.NewTestField("parent").WithType(api.TypezString),
			api.NewTestField("name").WithType(api.TypezString),
			api.NewTestField("child").WithMessageType(child).WithOptional(),
			api.NewTestField("optCount").WithType(api.TypezUint64).WithOptional(),
			api.NewTestField("kind").WithEnumType(kindEnum),
			oneofStrField,
			oneofNumField,
		).
		WithOneOfs(oneofStr, oneofNum)
	response := api.NewTestMessage("Response").WithPackage("test.v1")

	stringWithAdditionalBindings := api.NewTestMethod("StringOnly").
		WithInput(request).
		WithOutput(response).
		WithBindings(
			api.NewPathBindingBuilder().
				WithVerb("GET").
				WithPathTemplate(api.NewPathTemplateBuilder().
					WithLiteral("v1").
					WithVariableNamed("parent").
					WithLiteral("items").
					Build()).
				Build(),
			api.NewPathBindingBuilder().
				WithVerb("GET").
				WithPathTemplate(api.NewPathTemplateBuilder().
					WithLiteral("v1").
					WithVariableNamed("parent").
					WithLiteral("items").
					WithVariableNamed("name").
					Build()).
				Build(),
		)

	mixedTypes := api.NewTestMethod("MixedTypes").
		WithInput(request).
		WithOutput(response).
		WithVerb("GET").
		WithPathTemplate(api.NewPathTemplateBuilder().
			WithLiteral("v1").
			WithVariableNamed("parent").
			WithLiteral("children").
			WithVariableNamed("child", "id").
			WithLiteral("optCounts").
			WithVariableNamed("optCount").
			WithLiteral("kinds").
			WithVariableNamed("kind").
			WithLiteral("oneofStrs").
			WithVariableNamed("oneofStr").
			WithLiteral("oneofNums").
			WithVariableNamed("oneofNum").
			Build())

	mixedStream := api.NewTestMethod("MixedStream").
		WithInput(request).
		WithOutput(response).
		WithServerSideStreaming().
		WithVerb("GET").
		WithPathTemplate(api.NewPathTemplateBuilder().
			WithLiteral("v1").
			WithVariableNamed("parent").
			WithLiteral("optCounts").
			WithVariableNamed("optCount").
			Build())

	explicitPrecedence := api.NewTestMethod("ExplicitPrecedence").
		WithInput(request).
		WithOutput(response).
		WithVerb("GET").
		WithPathTemplate(api.NewPathTemplateBuilder().
			WithLiteral("v1").
			WithVariableNamed("name").
			Build()).
		WithRouting(api.NewTestRoutingInfo("routing_key", api.NewTestRoutingInfoVariant("parent")))

	noParams := api.NewTestMethod("NoParams").
		WithInput(request).
		WithOutput(response).
		WithVerb("GET").
		WithPathTemplate(api.NewPathTemplateBuilder().
			WithLiteral("v1").
			WithLiteral("items").
			Build())

	grpcOnly := api.NewTestMethod("GrpcOnly").
		WithInput(request).
		WithOutput(response)

	service := api.NewTestService("TestService").
		WithPackage("test.v1").
		WithMethods(stringWithAdditionalBindings, mixedTypes, mixedStream, explicitPrecedence, noParams, grpcOnly)

	model := api.NewTestAPI(
		[]*api.Message{request, child, response},
		[]*api.Enum{kindEnum},
		[]*api.Service{service},
	).WithPackageName("test.v1")
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	cfg := &parser.ModelConfig{
		SpecificationFormat: libconfig.SpecProtobuf,
		Codec:               tc.codec,
	}
	if err := Generate(t.Context(), model, outDir, cfg); err != nil {
		t.Fatal(err)
	}

	contentBytes, err := os.ReadFile(filepath.Join(outDir, tc.filePath))
	if err != nil {
		t.Fatal(err)
	}
	content := string(contentBytes)

	if strings.Contains(content, `.fold(String::new(), |b, p| b + "&" + &p)`) {
		t.Errorf("expected generated %s not to contain legacy fold expression", tc.filePath)
	}

	for _, test := range []struct {
		name     string
		startStr string
		kind     string
		want     string
	}{
		{
			name:     "string only deduplicates across additional bindings",
			startStr: "    async fn string_only(\n",
			kind:     "unary",
			want: `        let x_goog_request_params = gaxi::routing_parameter::format(&[
            Some(&req).map(|m| &m.parent).map(|s| s.as_str())
                .filter(|v| !v.is_empty())
                .map(|v| ("parent", v)),
            Some(&req).map(|m| &m.name).map(|s| s.as_str())
                .filter(|v| !v.is_empty())
                .map(|v| ("name", v)),
        ]);`,
		},
		{
			name:     "mixed types unary",
			startStr: "    async fn mixed_types(\n",
			kind:     "unary",
			want: `        let x_goog_request_params = {
            let var_opt_count = Some(&req).and_then(|m| m.opt_count.as_ref()).map(|v| v.to_string());
            let var_kind = Some(&req).map(|m| &m.kind).map(|v| v.to_string());
            let var_oneof_num = Some(&req).and_then(|m| m.oneof_num()).map(|v| v.to_string());
            gaxi::routing_parameter::format(&[
                Some(&req).map(|m| &m.parent).map(|s| s.as_str())
                    .filter(|v| !v.is_empty())
                    .map(|v| ("parent", v)),
                Some(&req).and_then(|m| m.child.as_ref()).map(|m| &m.id).map(|s| s.as_str())
                    .filter(|v| !v.is_empty())
                    .map(|v| ("child.id", v)),
                var_opt_count
                    .as_deref()
                    .filter(|v| !v.is_empty())
                    .map(|v| ("opt_count", v)),
                var_kind
                    .as_deref()
                    .filter(|v| !v.is_empty())
                    .map(|v| ("kind", v)),
                Some(&req).and_then(|m| m.oneof_str()).map(|s| s.as_str())
                    .filter(|v| !v.is_empty())
                    .map(|v| ("oneof_str", v)),
                var_oneof_num
                    .as_deref()
                    .filter(|v| !v.is_empty())
                    .map(|v| ("oneof_num", v)),
            ])
        };`,
		},
		{
			name:     "mixed types server streaming",
			startStr: "    async fn mixed_stream(\n",
			kind:     "stream",
			want: `        let x_goog_request_params = {
            let var_opt_count = Some(&req).and_then(|m| m.opt_count.as_ref()).map(|v| v.to_string());
            gaxi::routing_parameter::format(&[
                Some(&req).map(|m| &m.parent).map(|s| s.as_str())
                    .filter(|v| !v.is_empty())
                    .map(|v| ("parent", v)),
                var_opt_count
                    .as_deref()
                    .filter(|v| !v.is_empty())
                    .map(|v| ("opt_count", v)),
            ])
        };`,
		},
		{
			name:     "explicit routing takes precedence over implicit http",
			startStr: "    async fn explicit_precedence(\n",
			kind:     "unary",
			want: `        let x_goog_request_params = {
            use gaxi::routing_parameter::Segment;
            gaxi::routing_parameter::format(&[
                None
                .or_else(|| gaxi::routing_parameter::value(
                    Some(&req).map(|m| &m.parent).map(|s| s.as_str()),
                    &[ ],
                    &[ ],
                    &[ ],
                ))
                .map(|v| ("routing_key", v)),
            ])
        };`,
		},
		{
			name:     "no unique parameters",
			startStr: "    async fn no_params(\n",
			kind:     "unary",
			want:     `        let x_goog_request_params = String::new();`,
		},
		{
			name:     "nil path info",
			startStr: "    async fn grpc_only(\n",
			kind:     "unary",
			want:     `        let x_goog_request_params = String::new();`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			methodBody := extractBlock(t, content, test.startStr, "\n    }")
			endStr := tc.endMarkers[test.kind]
			raw := extractBlock(t, methodBody, "        let x_goog_request_params =", endStr)
			got := strings.TrimRight(strings.TrimSuffix(raw, endStr), " \n")
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateExplicitRoutingSegments(t *testing.T) {
	for _, test := range []struct {
		name     string
		filePath string
		codec    map[string]string
	}{
		{
			name:     "crate template",
			filePath: filepath.Join("src", "transport.rs"),
			codec: map[string]string{
				"package:wkt":       "source=google.protobuf,package=google-cloud-wkt",
				"default-transport": "grpc",
				"routing-required":  "true",
			},
		},
		{
			name:     "grpc-client template",
			filePath: "transport.rs",
			codec: map[string]string{
				"template-override": "templates/grpc-client",
				"package:wkt":       "source=google.protobuf,package=google-cloud-wkt",
				"routing-required":  "true",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			verifyExplicitRoutingSegments(t, test.filePath, test.codec)
		})
	}
}

func verifyExplicitRoutingSegments(t *testing.T, filePath string, codec map[string]string) {
	t.Helper()

	request := api.NewTestMessage("Request").
		WithPackage("test.v1").
		WithFields(
			api.NewTestField("name").WithType(api.TypezString),
			api.NewTestField("resource").WithType(api.TypezString),
			api.NewTestField("table_name").WithType(api.TypezString),
		)
	response := api.NewTestMessage("Response").WithPackage("test.v1")

	getFolder := api.NewTestMethod("GetFolder").
		WithInput(request).
		WithOutput(response).
		WithVerb("GET").
		WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").WithVariableNamed("name").Build()).
		WithRouting(api.NewRoutingInfoBuilder().
			WithName("bucket").
			WithVariant(api.NewRoutingInfoVariantBuilder().
				WithFieldPath("name").
				WithMatchingSegments("projects", "*", "buckets", "*").
				WithSuffixSegments("**").
				Build()).
			Build())

	testIamPermissions := api.NewTestMethod("TestIamPermissions").
		WithInput(request).
		WithOutput(response).
		WithVerb("POST").
		WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").WithVariableNamed("resource").Build()).
		WithRouting(api.NewRoutingInfoBuilder().
			WithName("bucket").
			WithVariants(
				api.NewRoutingInfoVariantBuilder().
					WithFieldPath("resource").
					WithMatchingSegments("projects", "*", "buckets", "*").
					WithSuffixSegments("managedFolders", "**").
					Build(),
				api.NewRoutingInfoVariantBuilder().
					WithFieldPath("resource").
					WithMatchingSegments("projects", "*", "buckets", "*").
					WithSuffixSegments("objects", "**").
					Build(),
				api.NewRoutingInfoVariantBuilder().
					WithFieldPath("resource").
					WithMatchingSegments("**").
					Build(),
			).
			Build())

	getInstanceTable := api.NewTestMethod("GetInstanceTable").
		WithInput(request).
		WithOutput(response).
		WithVerb("GET").
		WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").WithVariableNamed("table_name").Build()).
		WithRouting(api.NewRoutingInfoBuilder().
			WithName("instance_id").
			WithVariant(api.NewRoutingInfoVariantBuilder().
				WithFieldPath("table_name").
				WithPrefixSegments("projects", "*").
				WithMatchingSegments("instances", "*").
				WithSuffixSegments("**").
				Build()).
			Build())

	service := api.NewTestService("StorageControl").
		WithPackage("test.v1").
		WithMethods(getFolder, testIamPermissions, getInstanceTable)

	model := api.NewTestAPI([]*api.Message{request, response}, nil, []*api.Service{service}).
		WithPackageName("test.v1")
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	cfg := &parser.ModelConfig{
		SpecificationFormat: libconfig.SpecProtobuf,
		Codec:               codec,
	}
	if err := Generate(t.Context(), model, outDir, cfg); err != nil {
		t.Fatal(err)
	}

	contentBytes, err := os.ReadFile(filepath.Join(outDir, filePath))
	if err != nil {
		t.Fatal(err)
	}
	content := string(contentBytes)

	for _, test := range []struct {
		name     string
		startStr string
		want     string
	}{
		{
			name:     "get_folder emits TrailingMultiWildcard for singleton ** suffix",
			startStr: "    async fn get_folder(\n",
			want: `        let x_goog_request_params = {
            use gaxi::routing_parameter::Segment;
            gaxi::routing_parameter::format(&[
                None
                .or_else(|| gaxi::routing_parameter::value(
                    Some(&req).map(|m| &m.name).map(|s| s.as_str()),
                    &[ ],
                    &[Segment::Literal("projects/"),Segment::SingleWildcard,Segment::Literal("/buckets/"),Segment::SingleWildcard, ],
                    &[Segment::TrailingMultiWildcard, ],
                ))
                .map(|v| ("bucket", v)),
            ])
        };
        if x_goog_request_params.is_empty() {
            use google_cloud_gax::error::binding::BindingError;
            use gaxi::routing_parameter::Segment;
            use gaxi::path_parameter::PathMismatchBuilder;
            let mut paths = Vec::new();
            {
                let builder = PathMismatchBuilder::default();
                let builder = builder.maybe_add(
                    Some(&req).map(|m| &m.name).map(|s| s.as_str()),
                    &[Segment::Literal("projects/"),Segment::SingleWildcard,Segment::Literal("/buckets/"),Segment::SingleWildcard,Segment::TrailingMultiWildcard,
                    ],
                    "name",
                    "projects/*/buckets/*/**");
                paths.push(builder.build());
            }
            return Err(google_cloud_gax::error::Error::binding(BindingError { paths }))
        }`,
		},
		{
			name:     "test_iam_permissions preserves leading slash on suffix literals",
			startStr: "    async fn test_iam_permissions(\n",
			want: `        let x_goog_request_params = {
            use gaxi::routing_parameter::Segment;
            gaxi::routing_parameter::format(&[
                None
                .or_else(|| gaxi::routing_parameter::value(
                    Some(&req).map(|m| &m.resource).map(|s| s.as_str()),
                    &[ ],
                    &[Segment::Literal("projects/"),Segment::SingleWildcard,Segment::Literal("/buckets/"),Segment::SingleWildcard, ],
                    &[Segment::Literal("/managedFolders"),Segment::TrailingMultiWildcard, ],
                ))
                .or_else(|| gaxi::routing_parameter::value(
                    Some(&req).map(|m| &m.resource).map(|s| s.as_str()),
                    &[ ],
                    &[Segment::Literal("projects/"),Segment::SingleWildcard,Segment::Literal("/buckets/"),Segment::SingleWildcard, ],
                    &[Segment::Literal("/objects"),Segment::TrailingMultiWildcard, ],
                ))
                .or_else(|| gaxi::routing_parameter::value(
                    Some(&req).map(|m| &m.resource).map(|s| s.as_str()),
                    &[ ],
                    &[Segment::MultiWildcard, ],
                    &[ ],
                ))
                .map(|v| ("bucket", v)),
            ])
        };
        if x_goog_request_params.is_empty() {
            use google_cloud_gax::error::binding::BindingError;
            use gaxi::routing_parameter::Segment;
            use gaxi::path_parameter::PathMismatchBuilder;
            let mut paths = Vec::new();
            {
                let builder = PathMismatchBuilder::default();
                let builder = builder.maybe_add(
                    Some(&req).map(|m| &m.resource).map(|s| s.as_str()),
                    &[Segment::Literal("projects/"),Segment::SingleWildcard,Segment::Literal("/buckets/"),Segment::SingleWildcard,Segment::Literal("/managedFolders"),Segment::TrailingMultiWildcard,
                    ],
                    "resource",
                    "projects/*/buckets/*/managedFolders/**");
                paths.push(builder.build());
            }
            {
                let builder = PathMismatchBuilder::default();
                let builder = builder.maybe_add(
                    Some(&req).map(|m| &m.resource).map(|s| s.as_str()),
                    &[Segment::Literal("projects/"),Segment::SingleWildcard,Segment::Literal("/buckets/"),Segment::SingleWildcard,Segment::Literal("/objects"),Segment::TrailingMultiWildcard,
                    ],
                    "resource",
                    "projects/*/buckets/*/objects/**");
                paths.push(builder.build());
            }
            {
                let builder = PathMismatchBuilder::default();
                let builder = builder.maybe_add(
                    Some(&req).map(|m| &m.resource).map(|s| s.as_str()),
                    &[Segment::MultiWildcard,
                    ],
                    "resource",
                    "**");
                paths.push(builder.build());
            }
            return Err(google_cloud_gax::error::Error::binding(BindingError { paths }))
        }`,
		},
		{
			name:     "get_instance_table preserves prefix trailing slash and suffix wildcard",
			startStr: "    async fn get_instance_table(\n",
			want: `        let x_goog_request_params = {
            use gaxi::routing_parameter::Segment;
            gaxi::routing_parameter::format(&[
                None
                .or_else(|| gaxi::routing_parameter::value(
                    Some(&req).map(|m| &m.table_name).map(|s| s.as_str()),
                    &[Segment::Literal("projects/"),Segment::SingleWildcard,Segment::Literal("/"), ],
                    &[Segment::Literal("instances/"),Segment::SingleWildcard, ],
                    &[Segment::TrailingMultiWildcard, ],
                ))
                .map(|v| ("instance_id", v)),
            ])
        };
        if x_goog_request_params.is_empty() {
            use google_cloud_gax::error::binding::BindingError;
            use gaxi::routing_parameter::Segment;
            use gaxi::path_parameter::PathMismatchBuilder;
            let mut paths = Vec::new();
            {
                let builder = PathMismatchBuilder::default();
                let builder = builder.maybe_add(
                    Some(&req).map(|m| &m.table_name).map(|s| s.as_str()),
                    &[Segment::Literal("projects/"),Segment::SingleWildcard,Segment::Literal("/"),Segment::Literal("instances/"),Segment::SingleWildcard,Segment::TrailingMultiWildcard,
                    ],
                    "table_name",
                    "projects/*/instances/*/**");
                paths.push(builder.build());
            }
            return Err(google_cloud_gax::error::Error::binding(BindingError { paths }))
        }`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			methodBody := extractBlock(t, content, test.startStr, "\n    }")
			endStr := "\n        type TR ="
			raw := extractBlock(t, methodBody, "        let x_goog_request_params =", endStr)
			got := strings.TrimRight(strings.TrimSuffix(raw, endStr), " \n")
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
