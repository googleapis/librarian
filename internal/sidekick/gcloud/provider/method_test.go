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

package provider

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestIsCreate(t *testing.T) {
	for _, test := range []struct {
		name   string
		method *api.Method
		want   bool
	}{
		{"Name Prefix", &api.Method{Name: "CreateInstance"}, true},
		{"Name Mismatch", &api.Method{Name: "GetInstance"}, false},
		{"Verb Match", api.NewTestMethod("CreateInstance").WithVerb("POST"), true},
		{"Verb Mismatch", api.NewTestMethod("CreateInstance").WithVerb("GET"), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := IsCreate(test.method)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsGet(t *testing.T) {
	for _, test := range []struct {
		name   string
		method *api.Method
		want   bool
	}{
		{"Name Prefix", &api.Method{Name: "GetInstance"}, true},
		{"Name Mismatch", &api.Method{Name: "CreateInstance"}, false},
		{"Verb Match", api.NewTestMethod("GetInstance").WithVerb("GET"), true},
		{"Verb Mismatch", api.NewTestMethod("GetInstance").WithVerb("POST"), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := IsGet(test.method)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsList(t *testing.T) {
	for _, test := range []struct {
		name   string
		method *api.Method
		want   bool
	}{
		{"Name Prefix", &api.Method{Name: "ListInstances"}, true},
		{"Name Mismatch", &api.Method{Name: "GetInstance"}, false},
		{"Verb Match", api.NewTestMethod("ListInstances").WithVerb("GET"), true},
		{"Verb Mismatch", api.NewTestMethod("ListInstances").WithVerb("POST"), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := IsList(test.method)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsUpdate(t *testing.T) {
	for _, test := range []struct {
		name   string
		method *api.Method
		want   bool
	}{
		{"Name Prefix", &api.Method{Name: "UpdateInstance"}, true},
		{"Name Mismatch", &api.Method{Name: "GetInstance"}, false},
		{"Verb Match PATCH", api.NewTestMethod("UpdateInstance").WithVerb("PATCH"), true},
		{"Verb Match PUT", api.NewTestMethod("UpdateInstance").WithVerb("PUT"), true},
		{"Verb Mismatch", api.NewTestMethod("UpdateInstance").WithVerb("GET"), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := IsUpdate(test.method)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsDelete(t *testing.T) {
	for _, test := range []struct {
		name   string
		method *api.Method
		want   bool
	}{
		{"Name Prefix", &api.Method{Name: "DeleteInstance"}, true},
		{"Name Mismatch", &api.Method{Name: "GetInstance"}, false},
		{"Verb Match", api.NewTestMethod("DeleteInstance").WithVerb("DELETE"), true},
		{"Verb Mismatch", api.NewTestMethod("DeleteInstance").WithVerb("GET"), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := IsDelete(test.method)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetCommandName(t *testing.T) {
	v := "exportData"
	for _, test := range []struct {
		name   string
		method *api.Method
		want   string
	}{
		{"Standard Create", &api.Method{Name: "CreateInstance"}, "create"},
		{"Standard List", &api.Method{Name: "ListInstances"}, "list"},
		{"Standard Get", &api.Method{Name: "GetInstance"}, "describe"},
		{"Custom Verb in Path", api.NewTestMethod("ExportData").WithPathTemplate(&api.PathTemplate{Verb: &v}), "export_data"},
		{"Fallback to Name", &api.Method{Name: "ExportData"}, "export_data"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := GetCommandName(test.method)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestFindResourceMessage(t *testing.T) {
	instanceMsg := &api.Message{
		Name: "Instance",
	}
	for _, test := range []struct {
		name       string
		outputType *api.Message
		want       *api.Message
	}{
		{
			name: "Standard List Response",
			outputType: &api.Message{
				Fields: []*api.Field{
					{Name: "next_page_token", Typez: api.STRING_TYPE},
					{Name: "instances", Repeated: true, MessageType: instanceMsg},
				},
			},
			want: instanceMsg,
		},
		{
			name: "No Repeated Message",
			outputType: &api.Message{
				Fields: []*api.Field{
					{Name: "status", Typez: api.STRING_TYPE},
				},
			},
			want: nil,
		},
		{
			name:       "Nil Output Type",
			outputType: nil,
			want:       nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := FindResourceMessage(test.outputType)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetCommandName_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		method  *api.Method
		wantErr error
	}{
		{
			name:    "Nil Method",
			method:  nil,
			wantErr: errors.New("method cannot be nil"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, gotErr := GetCommandName(test.method)
			if test.wantErr != nil {
				if gotErr == nil {
					t.Fatalf("GetCommandName() returned nil error, want %v", test.wantErr)
				}
				if gotErr.Error() != test.wantErr.Error() {
					t.Errorf("GetCommandName() error = %q, want %q", gotErr.Error(), test.wantErr.Error())
				}
			} else if gotErr != nil {
				t.Errorf("GetCommandName() returned error %v, want nil", gotErr)
			}
		})
	}
}

func TestIsResourceMethod(t *testing.T) {
	for _, test := range []struct {
		name   string
		method *api.Method
		want   bool
	}{
		{"Standard Get", api.NewTestMethod("GetInstance").WithVerb("GET"), true},
		{"Standard List", api.NewTestMethod("ListInstances").WithVerb("GET"), false},
		{"Custom Resource", api.NewTestMethod("CustomInstance").WithPathTemplate(
			&api.PathTemplate{Segments: []api.PathSegment{*api.NewPathSegment().WithVariable(api.NewPathVariable("instance"))}},
		), true},
		{"Custom Collection", api.NewTestMethod("CustomCollection").WithPathTemplate(
			&api.PathTemplate{Segments: []api.PathSegment{*api.NewPathSegment().WithLiteral("instances")}},
		), false},
		{"Nil PathInfo", &api.Method{Name: "CustomMethod", PathInfo: nil}, false},
		{"Empty Bindings", &api.Method{Name: "CustomMethod", PathInfo: &api.PathInfo{Bindings: []*api.PathBinding{}}}, false},
		{"Nil PathTemplate", &api.Method{Name: "CustomMethod", PathInfo: &api.PathInfo{Bindings: []*api.PathBinding{{PathTemplate: nil}}}}, false},
		{"Empty Segments", &api.Method{Name: "CustomMethod", PathInfo: &api.PathInfo{Bindings: []*api.PathBinding{{PathTemplate: &api.PathTemplate{Segments: []api.PathSegment{}}}}}}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := IsResourceMethod(test.method); got != test.want {
				t.Errorf("mismatch (-want +got):\n%s", cmp.Diff(test.want, got))
			}
		})
	}
}

func TestIsCollectionMethod(t *testing.T) {
	for _, test := range []struct {
		name   string
		method *api.Method
		want   bool
	}{
		{"Standard Get", api.NewTestMethod("GetInstance").WithVerb("GET"), false},
		{"Standard List", api.NewTestMethod("ListInstances").WithVerb("GET"), true},
		{"Custom Resource", api.NewTestMethod("CustomInstance").WithPathTemplate(
			&api.PathTemplate{Segments: []api.PathSegment{*api.NewPathSegment().WithVariable(api.NewPathVariable("instance"))}},
		), false},
		{"Custom Collection", api.NewTestMethod("CustomCollection").WithPathTemplate(
			&api.PathTemplate{Segments: []api.PathSegment{*api.NewPathSegment().WithLiteral("instances")}},
		), true},
		{"Nil PathInfo", &api.Method{Name: "CustomMethod", PathInfo: nil}, false},
		{"Empty Bindings", &api.Method{Name: "CustomMethod", PathInfo: &api.PathInfo{Bindings: []*api.PathBinding{}}}, false},
		{"Nil PathTemplate", &api.Method{Name: "CustomMethod", PathInfo: &api.PathInfo{Bindings: []*api.PathBinding{{PathTemplate: nil}}}}, false},
		{"Empty Segments", &api.Method{Name: "CustomMethod", PathInfo: &api.PathInfo{Bindings: []*api.PathBinding{{PathTemplate: &api.PathTemplate{Segments: []api.PathSegment{}}}}}}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := IsCollectionMethod(test.method); got != test.want {
				t.Errorf("mismatch (-want +got):\n%s", cmp.Diff(test.want, got))
			}
		})
	}
}

func TestIsStandardMethod(t *testing.T) {
	for _, test := range []struct {
		name   string
		method *api.Method
		want   bool
	}{
		{"Get", api.NewTestMethod("GetInstance").WithVerb("GET"), true},
		{"List", api.NewTestMethod("ListInstances").WithVerb("GET"), true},
		{"Create", api.NewTestMethod("CreateInstance").WithVerb("POST"), true},
		{"Update", api.NewTestMethod("UpdateInstance").WithVerb("PATCH"), true},
		{"Delete", api.NewTestMethod("DeleteInstance").WithVerb("DELETE"), true},
		{"Custom", api.NewTestMethod("ExportInstance").WithVerb("POST"), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := IsStandardMethod(test.method)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsSingletonResourceMethod(t *testing.T) {
	model := &api.API{
		ResourceDefinitions: []*api.Resource{
			{
				Type: "test.googleapis.com/Singleton",
				Patterns: []api.ResourcePattern{
					[]api.PathSegment{
						*api.NewPathSegment().WithLiteral("projects"),
						*api.NewPathSegment().WithVariable(api.NewPathVariable("project")),
						*api.NewPathSegment().WithLiteral("singletonConfig"),
					},
				},
			},
			{
				Type: "test.googleapis.com/AdjacentLiterals",
				Patterns: []api.ResourcePattern{
					[]api.PathSegment{
						*api.NewPathSegment().WithLiteral("projects"),
						*api.NewPathSegment().WithVariable(api.NewPathVariable("project")),
						*api.NewPathSegment().WithLiteral("literal1"),
						*api.NewPathSegment().WithLiteral("literal2"),
						*api.NewPathSegment().WithVariable(api.NewPathVariable("instance")),
					},
				},
			},
			{
				Type: "test.googleapis.com/Standard",
				Patterns: []api.ResourcePattern{
					[]api.PathSegment{
						*api.NewPathSegment().WithLiteral("projects"),
						*api.NewPathSegment().WithVariable(api.NewPathVariable("project")),
						*api.NewPathSegment().WithLiteral("instances"),
						*api.NewPathSegment().WithVariable(api.NewPathVariable("instance")),
					},
				},
			},
		},
	}

	for _, test := range []struct {
		name   string
		method *api.Method
		model  *api.API
		want   bool
	}{
		{
			name:   "Nil Method",
			method: nil,
			model:  &api.API{},
			want:   false,
		},
		{
			name:   "Fallback: No Resource Found, Path Ends in Variable",
			method: api.NewTestMethod("Test").WithPathTemplate(&api.PathTemplate{Segments: parseResourcePattern("projects/{project}/instances/{instance}")}),
			model:  &api.API{},
			want:   false,
		},
		{
			name:   "Fallback: No Resource Found, Path Ends in Literal",
			method: api.NewTestMethod("Test").WithPathTemplate(&api.PathTemplate{Segments: parseResourcePattern("projects/{project}/singletonConfig")}),
			model:  &api.API{},
			want:   false, // Was true when fallback existed.
		},
		{
			name: "Resource Found: Singleton Pattern",
			method: func() *api.Method {
				m := api.NewTestMethod("GetSingleton")
				m.InputType = &api.Message{
					Fields: []*api.Field{{
						Name: "name",
						ResourceReference: &api.ResourceReference{
							Type: "test.googleapis.com/Singleton",
						},
					}},
				}
				return m
			}(),
			model: model,
			want:  true,
		},
		{
			name: "Resource Found: Adjacent Literals Pattern",
			method: func() *api.Method {
				m := api.NewTestMethod("GetAdjacent")
				m.InputType = &api.Message{
					Fields: []*api.Field{{
						Name: "name",
						ResourceReference: &api.ResourceReference{
							Type: "test.googleapis.com/AdjacentLiterals",
						},
					}},
				}
				return m
			}(),
			model: model,
			want:  true,
		},
		{
			name: "Resource Found: Standard Pattern",
			method: func() *api.Method {
				m := api.NewTestMethod("GetStandard")
				m.InputType = &api.Message{
					Fields: []*api.Field{{
						Name: "name",
						ResourceReference: &api.ResourceReference{
							Type: "test.googleapis.com/Standard",
						},
					}},
				}
				return m
			}(),
			model: model,
			want:  false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := IsSingletonResourceMethod(test.method, test.model)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetMethodHelpText(t *testing.T) {
	model := &api.API{
		ResourceDefinitions: []*api.Resource{
			{
				Type:     "test.googleapis.com/Instance",
				Plural:   "instances",
				Singular: "instance",
			},
		},
	}

	methodMock := func(name string) *api.Method {
		m := api.NewTestMethod(name)
		m.ID = "." + name
		m.InputType = &api.Message{
			Fields: []*api.Field{{
				Name: "name",
				ResourceReference: &api.ResourceReference{
					Type: "test.googleapis.com/Instance",
				},
			}},
		}
		return m
	}

	overrides := &Config{
		APIs: []API{
			{
				HelpText: &HelpTextRules{
					MethodRules: []*HelpTextRule{
						{
							Selector: "GetInstance",
							HelpText: &HelpTextElement{
								Brief:       "Override Brief",
								Description: "Override Desc",
								Examples:    []string{"Override Ex1", "Override Ex2"},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range []struct {
		name      string
		overrides *Config
		method    *api.Method
		want      HelpText
	}{
		{
			name:   "Describe Fallback",
			method: methodMock("GetInstance"),
			want: HelpText{
				Brief:       "Describe instances",
				Description: "Describe a instance",
				Examples:    "To describe the instance, run:\n\n    $ {command}",
			},
		},
		{
			name:   "List Fallback",
			method: methodMock("ListInstances"),
			want: HelpText{
				Brief:       "List instances",
				Description: "List instances",
				Examples:    "To list all instances, run:\n\n    $ {command}",
			},
		},
		{
			name:   "Create Fallback",
			method: methodMock("CreateInstance"),
			want: HelpText{
				Brief:       "Create instances",
				Description: "Create a instance",
				Examples:    "To create the instance, run:\n\n    $ {command}",
			},
		},
		{
			name:   "Delete Fallback",
			method: methodMock("DeleteInstance"),
			want: HelpText{
				Brief:       "Delete instances",
				Description: "Delete a instance",
				Examples:    "To delete the instance, run:\n\n    $ {command}",
			},
		},
		{
			name:   "Update Fallback",
			method: methodMock("UpdateInstance"),
			want: HelpText{
				Brief:       "Update instances",
				Description: "Update a instance",
				Examples:    "To update the instance, run:\n\n    $ {command}",
			},
		},
		{
			name:      "Override Found",
			overrides: overrides,
			method:    methodMock("GetInstance"),
			want: HelpText{
				Brief:       "Override Brief",
				Description: "Override Desc",
				Examples:    "Override Ex1\n\nOverride Ex2",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := GetMethodHelpText(test.overrides, test.method, model)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
