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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestPathExpression(t *testing.T) {
	for _, test := range []struct {
		name     string
		template *api.PathTemplate
		want     string
	}{
		{
			name: "literals only",
			template: api.NewPathTemplateBuilder().
				WithLiteral("v1").
				WithLiteral("operations").
				Build(),
			want: "/v1/operations",
		},
		{
			name: "with variable",
			template: api.NewPathTemplateBuilder().
				WithLiteral("v1").
				WithVariableNamed("name").
				Build(),
			want: "/v1/\\(pathVariable0)",
		},
		{
			name: "with multiple variables",
			template: api.NewPathTemplateBuilder().
				WithLiteral("v1").
				WithVariableNamed("name").WithLiteral("separator").WithVariableNamed("second").
				Build(),
			want: "/v1/\\(pathVariable0)/separator/\\(pathVariable1)",
		},
		{
			name: "with verb",
			template: api.NewPathTemplateBuilder().
				WithLiteral("v1").
				WithLiteral("operations").
				WithVerb("cancel").
				Build(),
			want: "/v1/operations:cancel",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := pathExpression(test.template)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPathVariables(t *testing.T) {
	requestMessage := api.NewTestMessage("CreateSecretRequest").
		WithPackage("google.cloud.secretmanager.v1").
		WithFields(
			api.NewTestField("name").WithType(api.TypezString),
			api.NewTestField("second").WithType(api.TypezString),
		)
	model := api.NewTestAPI([]*api.Message{requestMessage}, nil, nil)
	codec := newTestCodec(t, model, nil)
	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name     string
		template *api.PathTemplate
		want     []*pathVariable
		wantErr  bool
	}{
		{
			name:     "no variables",
			template: api.NewPathTemplateBuilder().WithLiteral("v1").Build(),
			want:     nil,
		},
		{
			name: "one variable",
			template: api.NewPathTemplateBuilder().
				WithLiteral("v1").
				WithVariableNamed("name").
				Build(),
			want: []*pathVariable{
				{
					Name:             "pathVariable0",
					Expression:       ".name as Swift.String?",
					Test:             "!pathVariable0.isEmpty",
					FieldPath:        "name",
					JSONFieldPath:    "name",
					MatchingSegments: []string{".singleWildcard"},
					TemplateString:   "*",
				},
			},
		},
		{
			name: "two variables",
			template: api.NewPathTemplateBuilder().
				WithLiteral("v1").
				WithVariableNamed("name").
				WithLiteral("sep").
				WithVariableNamed("second").
				Build(),
			want: []*pathVariable{
				{
					Name:             "pathVariable0",
					Expression:       ".name as Swift.String?",
					Test:             "!pathVariable0.isEmpty",
					FieldPath:        "name",
					JSONFieldPath:    "name",
					MatchingSegments: []string{".singleWildcard"},
					TemplateString:   "*",
				},
				{
					Name:             "pathVariable1",
					Expression:       ".second as Swift.String?",
					Test:             "!pathVariable1.isEmpty",
					FieldPath:        "second",
					JSONFieldPath:    "second",
					MatchingSegments: []string{".singleWildcard"},
					TemplateString:   "*",
				},
			},
		},
		{
			name: "error - lookup field missing",
			template: api.NewPathTemplateBuilder().
				WithLiteral("v1").
				WithVariableNamed("missing").
				Build(),
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := codec.pathVariables(requestMessage, test.template)
			if (err != nil) != test.wantErr {
				t.Fatalf("pathVariables() error = %v, wantErr %v", err, test.wantErr)
			}
			if test.wantErr {
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewPathVariable(t *testing.T) {
	secretMessage := api.NewTestMessage("Secret").
		WithPackage("google.cloud.secretmanager.v1").
		WithFields(
			api.NewTestField("name").WithType(api.TypezString),
			api.NewTestField("description").WithType(api.TypezString).WithOptional(),
		)

	oneof := api.NewTestOneOf("oneof_group").
		WithFields(api.NewTestField("oneof_field").WithType(api.TypezString))

	requestMessage := api.NewTestMessage("CreateSecretRequest").
		WithPackage("google.cloud.secretmanager.v1").
		WithFields(
			api.NewTestField("parent").WithType(api.TypezString),
			api.NewTestField("display_name").WithType(api.TypezString).WithOptional(),
			api.NewTestField("secret").WithMessageType(secretMessage).WithOptional(),
			api.NewTestField("data").WithType(api.TypezBytes),
			api.NewTestField("custom_name").
				WithType(api.TypezString).
				WithJSONName("custom_name"),
		).
		WithOneOfs(oneof)

	model := api.NewTestAPI([]*api.Message{secretMessage, requestMessage}, nil, nil)
	codec := newTestCodec(t, model, nil)
	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name     string
		message  *api.Message
		variable *api.PathVariable
		count    int
		want     *pathVariable
		wantErr  bool
	}{
		{
			name:     "non-optional string",
			message:  requestMessage,
			variable: api.NewPathVariableBuilder("parent").Build(),
			count:    0,
			want: &pathVariable{
				Name:          "pathVariable0",
				Expression:    ".parent as Swift.String?",
				Test:          "!pathVariable0.isEmpty",
				FieldPath:     "parent",
				JSONFieldPath: "parent",
			},
		},
		{
			name:     "optional string",
			message:  requestMessage,
			variable: api.NewPathVariableBuilder("display_name").Build(),
			count:    1,
			want: &pathVariable{
				Name:          "pathVariable1",
				Expression:    ".displayName",
				Test:          "!pathVariable1.isEmpty",
				FieldPath:     "display_name",
				JSONFieldPath: "displayName",
			},
		},
		{
			name:     "error - oneof field",
			message:  requestMessage,
			variable: api.NewPathVariableBuilder("oneof_field").Build(),
			count:    6,
			wantErr:  true,
		},
		{
			name:     "nested non-optional string",
			message:  requestMessage,
			variable: api.NewPathVariableBuilder("secret", "name").Build(),
			count:    2,
			want: &pathVariable{
				Name:          "pathVariable2",
				Expression:    ".secret.map({ $0.name })",
				Test:          "!pathVariable2.isEmpty",
				FieldPath:     "secret.name",
				JSONFieldPath: "secret.name",
			},
		},
		{
			name:     "nested optional string",
			message:  requestMessage,
			variable: api.NewPathVariableBuilder("secret", "description").Build(),
			count:    3,
			want: &pathVariable{
				Name:          "pathVariable3",
				Expression:    ".secret.flatMap({ $0.description })",
				Test:          "!pathVariable3.isEmpty",
				FieldPath:     "secret.description",
				JSONFieldPath: "secret.description",
			},
		},
		{
			name:     "explicit json_name",
			message:  requestMessage,
			variable: api.NewPathVariableBuilder("custom_name").Build(),
			count:    7,
			want: &pathVariable{
				Name:          "pathVariable7",
				Expression:    ".customName as Swift.String?",
				Test:          "!pathVariable7.isEmpty",
				FieldPath:     "custom_name",
				JSONFieldPath: "custom_name",
			},
		},
		{
			name:     "error - unsupported type bytes",
			message:  requestMessage,
			variable: api.NewPathVariableBuilder("data").Build(),
			count:    4,
			wantErr:  true,
		},
		{
			name:     "error - lookup field missing",
			message:  requestMessage,
			variable: api.NewPathVariableBuilder("missing").Build(),
			count:    5,
			wantErr:  true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := codec.newPathVariable(test.message, test.variable, test.count)
			if (err != nil) != test.wantErr {
				t.Fatalf("newPathVariable() error = %v, wantErr %v", err, test.wantErr)
			}
			if test.wantErr {
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRoutingParams(t *testing.T) {
	tableMessage := api.NewTestMessage("Table").
		WithPackage("google.storage.v2").
		WithFields(
			api.NewTestField("parent").WithType(api.TypezString),
		)
	requestMessage := api.NewTestMessage("GetBucketRequest").
		WithPackage("google.storage.v2").
		WithFields(
			api.NewTestField("name").WithType(api.TypezString),
			api.NewTestField("parent").WithType(api.TypezString),
			api.NewTestField("table").WithMessageType(tableMessage).WithOptional(),
		)

	model := api.NewTestAPI([]*api.Message{tableMessage, requestMessage}, nil, nil)
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}
	codec := newTestCodec(t, model, nil)
	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name         string
		routingInfos []*api.RoutingInfo
		want         []*routingParam
	}{
		{
			name: "explicit routing info with custom name",
			routingInfos: []*api.RoutingInfo{
				api.NewRoutingInfoBuilder().
					WithName("bucket").
					WithVariants(
						api.NewRoutingInfoVariantBuilder().
							WithFieldPath("name").
							Build(),
					).
					Build(),
			},
			want: []*routingParam{
				{
					RoutingKey: "bucket",
					Variants: []*routingParamVariant{
						{
							FieldSteps: []*routingFieldStep{{PropertyName: "name"}},
							Last:       true,
						},
					},
				},
			},
		},
		{
			name: "explicit routing info without custom name",
			routingInfos: []*api.RoutingInfo{
				api.NewRoutingInfoBuilder().
					WithName("").
					WithVariants(
						api.NewRoutingInfoVariantBuilder().
							WithFieldPath("parent").
							Build(),
					).
					Build(),
			},
			want: []*routingParam{
				{
					RoutingKey: "parent",
					Variants: []*routingParamVariant{
						{
							FieldSteps: []*routingFieldStep{{PropertyName: "parent"}},
							Last:       true,
						},
					},
				},
			},
		},
		{
			name: "routing info with pattern and suffix",
			routingInfos: []*api.RoutingInfo{
				api.NewRoutingInfoBuilder().
					WithName("bucket").
					WithVariants(
						api.NewRoutingInfoVariantBuilder().
							WithFieldPath("name").
							WithMatchingSegments("projects", "*", "buckets", "*").
							WithSuffixSegments("**").
							Build(),
					).
					Build(),
			},
			want: []*routingParam{
				{
					RoutingKey: "bucket",
					Variants: []*routingParamVariant{
						{
							FieldSteps:       []*routingFieldStep{{PropertyName: "name"}},
							MatchingSegments: []string{`.literal("projects/")`, `.singleWildcard`, `.literal("/buckets/")`, `.singleWildcard`},
							SuffixSegments:   []string{`.trailingMultiWildcard`},
							Last:             true,
						},
					},
				},
			},
		},
		{
			name: "routing info with multiple fallback variants across different fields",
			routingInfos: []*api.RoutingInfo{
				api.NewRoutingInfoBuilder().
					WithName("routing_id").
					WithVariants(
						api.NewRoutingInfoVariantBuilder().
							WithFieldPath("name").
							WithMatchingSegments("projects", "*").
							Build(),
						api.NewRoutingInfoVariantBuilder().
							WithFieldPath("parent").
							WithMatchingSegments("**").
							Build(),
					).
					Build(),
			},
			want: []*routingParam{
				{
					RoutingKey: "routing_id",
					Variants: []*routingParamVariant{
						{
							FieldSteps:       []*routingFieldStep{{PropertyName: "name"}},
							MatchingSegments: []string{`.literal("projects/")`, `.singleWildcard`},
							Last:             false,
						},
						{
							FieldSteps:       []*routingFieldStep{{PropertyName: "parent"}},
							MatchingSegments: []string{`.multiWildcard`},
							Last:             true,
						},
					},
				},
			},
		},
		{
			name: "routing info with literal prefix and nested field path",
			routingInfos: []*api.RoutingInfo{
				api.NewRoutingInfoBuilder().
					WithName("table").
					WithVariants(
						api.NewRoutingInfoVariantBuilder().
							WithFieldPath("table", "parent").
							WithPrefixSegments("projects").
							WithMatchingSegments("*").
							WithSuffixSegments("tables", "*").
							Build(),
					).
					Build(),
			},
			want: []*routingParam{
				{
					RoutingKey: "table",
					Variants: []*routingParamVariant{
						{
							FieldSteps: []*routingFieldStep{
								{PropertyName: "table"},
								{PropertyName: "parent", ChainOptional: true},
							},
							IsOptional:       true,
							NeedsParens:      true,
							PrefixSegments:   []string{`.literal("projects/")`},
							MatchingSegments: []string{`.singleWildcard`},
							SuffixSegments:   []string{`.literal("/tables/")`, `.singleWildcard`},
							Last:             true,
						},
					},
				},
			},
		},
		{
			name: "routing info with wildcard prefix",
			routingInfos: []*api.RoutingInfo{
				api.NewRoutingInfoBuilder().
					WithName("location").
					WithVariants(
						api.NewRoutingInfoVariantBuilder().
							WithFieldPath("name").
							WithPrefixSegments("projects", "*").
							WithMatchingSegments("locations", "*").
							Build(),
					).
					Build(),
			},
			want: []*routingParam{
				{
					RoutingKey: "location",
					Variants: []*routingParamVariant{
						{
							FieldSteps:       []*routingFieldStep{{PropertyName: "name"}},
							PrefixSegments:   []string{`.literal("projects/")`, `.singleWildcard`, `.literal("/")`},
							MatchingSegments: []string{`.literal("locations/")`, `.singleWildcard`},
							Last:             true,
						},
					},
				},
			},
		},
		{
			name:         "empty routing annotation disables routing",
			routingInfos: []*api.RoutingInfo{},
			want:         nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := codec.routingParamsFromRouting(requestMessage, test.routingInfos)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRoutingParamsFromPathTemplate(t *testing.T) {
	requestMessage := api.NewTestMessage("GetBucketRequest").
		WithPackage("google.storage.v2").
		WithFields(
			api.NewTestField("name").WithType(api.TypezString),
		)
	model := api.NewTestAPI([]*api.Message{requestMessage}, nil, nil)
	codec := newTestCodec(t, model, nil)
	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name     string
		template *api.PathTemplate
		want     []*routingParam
	}{
		{
			name:     "nil template",
			template: nil,
			want:     nil,
		},
		{
			name: "template with variable",
			template: api.NewPathTemplateBuilder().
				WithLiteral("v2").
				WithVariableNamed("name").
				Build(),
			want: []*routingParam{newTestMultiWildcardRoutingParam("name", []*routingFieldStep{{PropertyName: "name"}}, false)},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := codec.routingParamsFromPathTemplate(requestMessage, test.template)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRoutingParamsFromBindings(t *testing.T) {
	bookMessage := api.NewTestMessage("Book").
		WithPackage("test.v1").
		WithFields(
			api.NewTestField("name").WithType(api.TypezString),
		)
	requestMessage := api.NewTestMessage("RoutingBindingsRequest").
		WithPackage("test.v1").
		WithFields(
			api.NewTestField("name").WithType(api.TypezString),
			api.NewTestField("parent").WithType(api.TypezString),
			api.NewTestField("destination_parent").WithType(api.TypezString),
			api.NewTestField("shelf").WithType(api.TypezString),
			api.NewTestField("book").WithMessageType(bookMessage).WithOptional(),
		)
	model := api.NewTestAPI([]*api.Message{bookMessage, requestMessage}, nil, nil)
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}
	c := newTestCodec(t, model, nil)
	if err := c.annotateModel(); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name     string
		bindings []*api.PathBinding
		want     []*routingParam
	}{
		{
			name:     "empty bindings",
			bindings: nil,
			want:     nil,
		},
		{
			name: "literal-only bindings",
			bindings: []*api.PathBinding{
				newTestPathBinding("GET", api.NewPathTemplateBuilder().WithLiteral("v1").WithLiteral("items").Build()),
			},
			want: nil,
		},
		{
			name: "nil binding and nil template are skipped",
			bindings: []*api.PathBinding{
				nil,
				api.NewPathBindingBuilder().WithVerb("GET").Build(),
				newTestPathBinding("GET", api.NewPathTemplateBuilder().WithLiteral("v1").WithVariableNamed("parent").Build()),
			},
			want: []*routingParam{newTestMultiWildcardRoutingParam("parent", []*routingFieldStep{{PropertyName: "parent"}}, false)},
		},
		{
			// CloudBilling.ListBillingAccounts: GET /v1/billingAccounts + GET /v1/{parent=organizations/*}/billingAccounts
			name: "primary literal-only with variable in additional_bindings",
			bindings: []*api.PathBinding{
				newTestPathBinding("GET", api.NewPathTemplateBuilder().WithLiteral("v1").WithLiteral("billingAccounts").Build()),
				newTestPathBinding("GET", api.NewPathTemplateBuilder().WithLiteral("v1").WithVariableNamed("parent").WithLiteral("billingAccounts").Build()),
			},
			want: []*routingParam{newTestMultiWildcardRoutingParam("parent", []*routingFieldStep{{PropertyName: "parent"}}, false)},
		},
		{
			// CloudBilling.MoveBillingAccount: POST /v1/{name=billingAccounts/*}:move + GET /v1/{destination_parent=organizations/*}/{name=billingAccounts/*}:move
			name: "overlapping and secondary-only variables preserve first-seen order",
			bindings: []*api.PathBinding{
				newTestPathBinding("POST", api.NewPathTemplateBuilder().WithLiteral("v1").WithVariableNamed("name").WithVerb("move").Build()),
				newTestPathBinding("GET", api.NewPathTemplateBuilder().WithLiteral("v1").WithVariableNamed("destination_parent").WithVariableNamed("name").WithVerb("move").Build()),
			},
			want: []*routingParam{
				newTestMultiWildcardRoutingParam("name", []*routingFieldStep{{PropertyName: "name"}}, false),
				newTestMultiWildcardRoutingParam("destination_parent", []*routingFieldStep{{PropertyName: "destinationParent"}}, false),
			},
		},
		{
			// StorageControl.SummarizeIntelligenceFindings: repeated {parent=...} with distinct resource sub-patterns across 3 bindings
			name: "distinct sub-patterns on same variable deduplicated",
			bindings: []*api.PathBinding{
				newTestPathBinding("GET", api.NewPathTemplateBuilder().
					WithLiteral("v2").
					WithVariable(api.NewPathVariableBuilder("parent").WithLiteral("projects").WithMatch().Build()).
					WithLiteral("intelligenceFindings").
					WithVerb("summarize").Build()),
				newTestPathBinding("GET", api.NewPathTemplateBuilder().
					WithLiteral("v2").
					WithVariable(api.NewPathVariableBuilder("parent").WithLiteral("folders").WithMatch().Build()).
					WithLiteral("intelligenceFindings").
					WithVerb("summarize").Build()),
				newTestPathBinding("GET", api.NewPathTemplateBuilder().
					WithLiteral("v2").
					WithVariable(api.NewPathVariableBuilder("parent").WithLiteral("organizations").WithMatch().Build()).
					WithLiteral("intelligenceFindings").
					WithVerb("summarize").Build()),
			},
			want: []*routingParam{newTestMultiWildcardRoutingParam("parent", []*routingFieldStep{{PropertyName: "parent"}}, false)},
		},
		{
			name: "nested field path deduplication",
			bindings: []*api.PathBinding{
				newTestPathBinding("PATCH", api.NewPathTemplateBuilder().WithLiteral("v1").WithVariableNamed("book", "name").Build()),
				newTestPathBinding("PATCH", api.NewPathTemplateBuilder().WithLiteral("v1").WithVariableNamed("shelf").WithVariableNamed("book", "name").Build()),
			},
			want: []*routingParam{
				newTestMultiWildcardRoutingParam(
					"book.name",
					[]*routingFieldStep{
						{PropertyName: "book"},
						{PropertyName: "name", ChainOptional: true},
					},
					true,
				),
				newTestMultiWildcardRoutingParam("shelf", []*routingFieldStep{{PropertyName: "shelf"}}, false),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := c.routingParamsFromBindings(requestMessage, test.bindings)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func newTestPathBinding(verb string, template *api.PathTemplate) *api.PathBinding {
	return api.NewPathBindingBuilder().WithVerb(verb).WithPathTemplate(template).Build()
}

func newTestMultiWildcardRoutingParam(key string, steps []*routingFieldStep, isOptional bool) *routingParam {
	return &routingParam{
		RoutingKey: key,
		Variants: []*routingParamVariant{
			{
				FieldSteps:       steps,
				IsOptional:       isOptional,
				NeedsParens:      isOptional && len(steps) > 1,
				MatchingSegments: []string{".multiWildcard"},
				Last:             true,
			},
		},
	}
}
