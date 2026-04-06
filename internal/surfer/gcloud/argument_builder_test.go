// Copyright 2025 Google LLC
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

package gcloud

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/surfer/gcloud/provider"
)

func TestNewArgument(t *testing.T) {
	model := &api.API{
		ResourceDefinitions: []*api.Resource{
			{
				Type: "test.googleapis.com/Network",
				Patterns: []api.ResourcePattern{
					{*api.NewPathSegment().WithLiteral("projects"), *api.NewPathSegment().WithVariable(api.NewPathVariable("project").WithMatch()), *api.NewPathSegment().WithLiteral("networks"), *api.NewPathSegment().WithVariable(api.NewPathVariable("network").WithMatch())},
				},
			},
		},
	}
	service := &api.Service{DefaultHost: "test.googleapis.com"}

	for _, test := range []struct {
		name      string
		field     *api.Field
		apiField  string
		method    *api.Method
		overrides *provider.Config
		want      Argument
	}{
		{
			name:     "String Field",
			field:    api.NewTestField("description").WithType(api.STRING_TYPE).WithBehavior(api.FIELD_BEHAVIOR_OPTIONAL),
			apiField: "description",
			method:   api.NewTestMethod("CreateInstance"),
			want: Argument{
				ArgName:  "description",
				APIField: "description",
				Type:     "str",
				HelpText: "Value for the `description` field.",
				Required: false,
				Repeated: false,
			},
		},
		{
			name:     "Resource Reference Field",
			field:    api.NewTestField("network").WithResourceReference("test.googleapis.com/Network"),
			apiField: "network",
			method:   api.NewTestMethod("CreateInstance"),
			want: Argument{
				ArgName:  "network",
				APIField: "network",
				HelpText: "Value for the `network` field.",
				ResourceSpec: &ResourceSpec{
					Name:       "network",
					PluralName: "networks",
					Collection: "test.projects.networks",
					Attributes: []Attribute{
						{AttributeName: "project", ParameterName: "projectsId", Help: "The project id of the {resource} resource.", Property: "core/project"},
						{AttributeName: "network", ParameterName: "networksId", Help: "The network id of the {resource} resource."},
					},
					DisableAutoCompleters: true,
				},
				ResourceMethodParams: map[string]string{"network": "{__relative_name__}"},
			},
		},
		{
			name: "Help Text Override",
			field: func() *api.Field {
				f := api.NewTestField("foo").WithType(api.STRING_TYPE)
				f.ID = "test.foo"
				return f
			}(),
			overrides: &provider.Config{
				APIs: []provider.API{
					{
						HelpText: &provider.HelpTextRules{
							FieldRules: []*provider.HelpTextRule{
								{Selector: "test.foo", HelpText: &provider.HelpTextElement{Brief: "Override Foo"}},
							},
						},
					},
				},
			},
			apiField: "foo",
			method:   api.NewTestMethod("CreateInstance"),
			want: Argument{
				ArgName:  "foo",
				APIField: "foo",
				Type:     "str",
				HelpText: "Override Foo",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			overrides := test.overrides
			if overrides == nil {
				overrides = &provider.Config{}
			}
			got, err := newArgumentBuilder(test.method, overrides, model, service, test.field, test.apiField).build()
			if err != nil {
				t.Errorf("newArgument(%s) unexpected error: %v", test.name, err)
				return
			}
			if got == nil {
				t.Errorf("newArgument(%s) returned nil, want non-nil", test.name)
				return
			}
			if diff := cmp.Diff(test.want, *got); diff != "" {
				t.Errorf("newArgument() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsIgnored(t *testing.T) {
	for _, test := range []struct {
		name   string
		field  *api.Field
		method *api.Method
		want   bool
	}{
		{
			name:  "Primary Resource ID (Create)",
			field: api.NewTestField("thing_id").WithType(api.STRING_TYPE),
			method: api.NewTestMethod("CreateThing").WithVerb("POST").WithInput(
				api.NewTestMessage("CreateRequest").WithFields(
					api.NewTestField("thing").WithType(api.MESSAGE_TYPE).WithMessageType(
						api.NewTestMessage("Thing").WithFields(
							api.NewTestField("name").WithType(api.STRING_TYPE),
						).WithResource(api.NewTestResource("test.googleapis.com/Thing")),
					),
				),
			),
			want: false,
		},
		{
			name:  "Update Mask",
			field: api.NewTestField("update_mask").WithType(api.MESSAGE_TYPE),
			method: api.NewTestMethod("UpdateThing").WithVerb("PATCH").WithInput(
				api.NewTestMessage("UpdateRequest").WithFields(
					api.NewTestField("update_mask").WithType(api.MESSAGE_TYPE),
				),
			),
			want: true,
		},
		{
			name: "Immutable field in Update",
			field: func() *api.Field {
				f := api.NewTestField("immutable_field").WithType(api.STRING_TYPE)
				f.Behavior = []api.FieldBehavior{api.FIELD_BEHAVIOR_IMMUTABLE}
				return f
			}(),
			method: api.NewTestMethod("UpdateThing").WithVerb("PATCH"),
			want:   true,
		},
		{
			name:  "Page Size (List)",
			field: api.NewTestField("page_size"),
			method: func() *api.Method {
				m := api.NewTestMethod("ListThings").WithVerb("GET").WithOutput(
					api.NewTestMessage("ListResponse").WithFields(
						api.NewTestField("things").WithType(api.MESSAGE_TYPE).WithRepeated(),
						api.NewTestField("next_page_token").WithType(api.STRING_TYPE),
					),
				)
				m.OutputType.Pagination = &api.PaginationInfo{
					PageableItem: m.OutputType.Fields[0],
				}
				return m
			}(),
			want: true,
		},
		{
			name:   "Immutable Field (Update)",
			field:  api.NewTestField("immutable").WithBehavior(api.FIELD_BEHAVIOR_IMMUTABLE),
			method: api.NewTestMethod("UpdateThing").WithVerb("PATCH"),
			want:   true,
		},
		{
			name:   "Output Only Field",
			field:  api.NewTestField("output_only").WithBehavior(api.FIELD_BEHAVIOR_OUTPUT_ONLY),
			method: api.NewTestMethod("CreateThing").WithVerb("POST"),
			want:   true,
		},
		{
			name:   "Regular Field",
			field:  api.NewTestField("description"),
			method: api.NewTestMethod("CreateThing").WithVerb("POST"),
			want:   false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			builder := newArgumentBuilder(test.method, nil, nil, nil, test.field, "")
			got := builder.isIgnored()
			if got != test.want {
				t.Errorf("isIgnored() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestNewPrimaryResourceArgument(t *testing.T) {
	for _, test := range []struct {
		name         string
		field        *api.Field
		method       *api.Method
		resourceDefs []*api.Resource
		want         Argument
	}{
		{
			name: "Create Instance (Positional)",
			field: &api.Field{
				Name:          "thing_id",
				Typez:         api.STRING_TYPE,
				Documentation: "The thing to create.",
			},
			method: func() *api.Method {
				m := api.NewTestMethod("CreateThing").WithVerb("POST").WithInput(
					api.NewTestMessage("CreateRequest").WithFields(
						api.NewTestField("thing").WithType(api.MESSAGE_TYPE).WithMessageType(
							api.NewTestMessage("Thing").WithFields(
								&api.Field{
									Name:          "name",
									Typez:         api.STRING_TYPE,
									Documentation: "The thing to create.",
								},
							).WithResource(&api.Resource{
								Type:     "test.googleapis.com/Thing",
								Singular: "thing",
								Plural:   "things",
								Patterns: []api.ResourcePattern{
									{
										*api.NewPathSegment().WithLiteral("projects"),
										*api.NewPathSegment().WithVariable(api.NewPathVariable("project").WithMatch()),
										*api.NewPathSegment().WithLiteral("things"),
										*api.NewPathSegment().WithVariable(api.NewPathVariable("thing").WithMatch()),
									},
								},
							}),
						),
					),
				)
				return m
			}(),
			resourceDefs: []*api.Resource{
				{
					Type:     "test.googleapis.com/Thing",
					Singular: "thing",
					Plural:   "things",
					Patterns: []api.ResourcePattern{
						{
							*api.NewPathSegment().WithLiteral("projects"),
							*api.NewPathSegment().WithVariable(api.NewPathVariable("project").WithMatch()),
							*api.NewPathSegment().WithLiteral("things"),
							*api.NewPathSegment().WithVariable(api.NewPathVariable("thing").WithMatch()),
						},
					},
				},
			},
			want: Argument{
				HelpText:          "The thing to create.",
				IsPositional:      true,
				IsPrimaryResource: true,
				Required:          true,
				RequestIDField:    "thingId",
				ResourceSpec: &ResourceSpec{
					Name:       "thing",
					PluralName: "things",
					Collection: "test.projects.things",
					Attributes: []Attribute{
						{
							ParameterName: "projectsId",
							AttributeName: "project",
							Help:          "The project id of the {resource} resource.",
							Property:      "core/project",
						},
						{
							ParameterName: "thingsId",
							AttributeName: "thing",
							Help:          "The thing id of the {resource} resource.",
						},
					},
				},
			},
		},
		{
			name: "List Instances (Not Positional, Parent)",
			field: func() *api.Field {
				f := api.NewTestField("parent").WithType(api.STRING_TYPE).WithResourceReference("test.googleapis.com/Thing")
				f.ResourceReference.ChildType = "test.googleapis.com/Thing"
				f.Documentation = "The parent of the resource."
				return f
			}(),
			method: func() *api.Method {
				m := api.NewTestMethod("ListThings").WithVerb("GET").WithInput(
					api.NewTestMessage("ListRequest").WithFields(
						api.NewTestField("parent").WithType(api.STRING_TYPE).WithResourceReference("test.googleapis.com/Thing"),
					),
				)
				m.InputType.Fields[0].ResourceReference.ChildType = "test.googleapis.com/Thing"
				return m
			}(),
			resourceDefs: []*api.Resource{
				{
					Type:     "test.googleapis.com/Thing",
					Singular: "thing",
					Plural:   "things",
					Patterns: []api.ResourcePattern{
						{
							*api.NewPathSegment().WithLiteral("projects"),
							*api.NewPathSegment().WithVariable(api.NewPathVariable("project").WithMatch()),
							*api.NewPathSegment().WithLiteral("things"),
							*api.NewPathSegment().WithVariable(api.NewPathVariable("thing").WithMatch()),
						},
					},
				},
			},
			want: Argument{
				HelpText:          "The parent of the resource.",
				IsPositional:      false,
				IsPrimaryResource: true,
				Required:          true,
				ResourceSpec: &ResourceSpec{
					Name:       "project",
					PluralName: "projects",
					Collection: "test.projects",
					Attributes: []Attribute{
						{
							ParameterName: "projectsId",
							AttributeName: "project",
							Help:          "The project id of the {resource} resource.",
							Property:      "core/project",
						},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			service := api.NewTestService("TestService").WithPackage("google.cloud.test.v1")
			service.DefaultHost = "test.googleapis.com"
			model := api.NewTestAPI([]*api.Message{}, nil, []*api.Service{service})
			model.ResourceDefinitions = test.resourceDefs

			test.method.Service = service
			test.method.Model = model

			var idField *api.Field
			if provider.IsCreate(test.method) {
				idField = test.field
			}
			got := newArgumentBuilder(test.method, nil, model, service, test.field, "").buildPrimaryResource(idField)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("newPrimaryResourceArgument() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestArgumentBuilder_Build(t *testing.T) {
	service := api.NewTestService("TestService")
	model := api.NewTestAPI([]*api.Message{}, nil, []*api.Service{service})

	createMethod := api.NewTestMethod("CreateThing").WithVerb("POST").WithInput(
		api.NewTestMessage("CreateRequest").WithFields(
			api.NewTestField("thing_id").WithType(api.STRING_TYPE),
		),
	)
	createMethod.Service = service
	createMethod.Model = model

	for _, test := range []struct {
		name    string
		field   *api.Field
		prefix  string
		want    *Argument
		wantErr bool
	}{
		{
			name:   "Skips skipped fields",
			field:  api.NewTestField("update_mask").WithType(api.MESSAGE_TYPE),
			prefix: "update_mask",
			want:   nil,
		},
		{
			name:   "Handles Simple String Field",
			field:  api.NewTestField("display_name").WithType(api.STRING_TYPE),
			prefix: "displayName",
			want: &Argument{
				ArgName:  "display-name",
				APIField: "displayName",
				HelpText: "Value for the `display-name` field.",
				Required: false,
				Repeated: false,
				Type:     "str",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			builder := newArgumentBuilder(createMethod, &provider.Config{}, model, service, test.field, test.prefix)
			got, err := builder.build()
			if (err != nil) != test.wantErr {
				t.Fatalf("build() error = %v, wantErr %v", err, test.wantErr)
			}
			if diff := cmp.Diff(test.want, got, cmpopts.IgnoreUnexported(Argument{}), cmpopts.IgnoreFields(Argument{}, "ResourceSpec")); diff != "" {
				t.Errorf("build() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewResourceReferenceSpec(t *testing.T) {
	service := api.NewTestService("TestService")
	service.DefaultHost = "test.googleapis.com"

	model := &api.API{
		ResourceDefinitions: []*api.Resource{
			{
				Type: "test.googleapis.com/OtherThing",
				Patterns: []api.ResourcePattern{
					{
						*api.NewPathSegment().WithLiteral("projects"),
						*api.NewPathSegment().WithVariable(api.NewPathVariable("project").WithMatch()),
						*api.NewPathSegment().WithLiteral("otherThings"),
						*api.NewPathSegment().WithVariable(api.NewPathVariable("other_thing").WithMatch()),
					},
				},
			},
		},
	}

	for _, test := range []struct {
		name  string
		field *api.Field
		want  *ResourceSpec
	}{
		{
			name:  "Handles valid resource reference",
			field: api.NewTestField("other_thing").WithResourceReference("test.googleapis.com/OtherThing"),
			want: &ResourceSpec{
				Name:                  "other_thing",
				PluralName:            "otherThings",
				Collection:            "test.projects.otherThings",
				DisableAutoCompleters: true,
				Attributes: []Attribute{
					{ParameterName: "projectsId", AttributeName: "project", Help: "The project id of the {resource} resource.", Property: "core/project"},
					{ParameterName: "otherThingsId", AttributeName: "other_thing", Help: "The other_thing id of the {resource} resource."},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := newArgumentBuilder(nil, nil, model, service, test.field, "").resourceReferenceSpec()
			if err != nil {
				t.Fatalf("newResourceReferenceSpec() unexpected error = %v", err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("newResourceReferenceSpec() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewResourceReferenceSpec_Error(t *testing.T) {
	service := api.NewTestService("TestService")

	for _, test := range []struct {
		name  string
		field *api.Field
	}{
		{
			name:  "Fails for missing resource definition",
			field: api.NewTestField("unknown").WithResourceReference("unknown.googleapis.com/Unknown"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := newArgumentBuilder(nil, nil, &api.API{}, service, test.field, "").resourceReferenceSpec()
			if err == nil {
				t.Fatalf("newResourceReferenceSpec() expected error, got nil")
			}
		})
	}
}
