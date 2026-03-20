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
			got := (&MethodAdapter{Method: test.method}).Type() == MethodTypeCreate
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
			got := (&MethodAdapter{Method: test.method}).Type() == MethodTypeGet
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
			got := (&MethodAdapter{Method: test.method}).Type() == MethodTypeList
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
			got := (&MethodAdapter{Method: test.method}).Type() == MethodTypeUpdate
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
			got := (&MethodAdapter{Method: test.method}).Type() == MethodTypeDelete
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
			got, err := (&MethodAdapter{Method: test.method}).GetCommandName()
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
			_, gotErr := (&MethodAdapter{Method: test.method}).GetCommandName()
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
			if got := (&MethodAdapter{Method: test.method}).isResourceMethod(); got != test.want {
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
			if got := (&MethodAdapter{Method: test.method}).isCollectionMethod(); got != test.want {
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
			got := (&MethodAdapter{Method: test.method}).IsStandardMethod()
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsPrimaryResource(t *testing.T) {
	for _, test := range []struct {
		name   string
		field  *api.Field
		method *api.Method
		want   bool
	}{
		{
			name:  "Create Method - Primary Resource ID",
			field: &api.Field{Name: "instance_id"},
			method: &api.Method{
				Name: "CreateInstance",
				InputType: &api.Message{
					Fields: []*api.Field{
						{
							MessageType: &api.Message{
								Name: "Instance",
								Resource: &api.Resource{
									Type: "example.googleapis.com/Instance",
								},
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name:  "Create Method - Not Primary Resource",
			field: &api.Field{Name: "parent"},
			method: &api.Method{
				Name: "CreateInstance",
				InputType: &api.Message{
					Fields: []*api.Field{
						{
							MessageType: &api.Message{
								Name: "Instance",
								Resource: &api.Resource{
									Type: "example.googleapis.com/Instance",
								},
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name:  "Get Method - Primary Resource Name",
			field: &api.Field{Name: "name"},
			method: &api.Method{
				Name: "GetInstance",
				InputType: &api.Message{
					Fields: []*api.Field{{Name: "name"}},
				},
			},
			want: true,
		},
		{
			name:  "Delete Method - Primary Resource Name",
			field: &api.Field{Name: "name"},
			method: &api.Method{
				Name: "DeleteInstance",
				InputType: &api.Message{
					Fields: []*api.Field{{Name: "name"}},
				},
			},
			want: true,
		},
		{
			name:  "Update Method - Primary Resource Name",
			field: &api.Field{Name: "name"},
			method: &api.Method{
				Name: "UpdateInstance",
				InputType: &api.Message{
					Fields: []*api.Field{{Name: "name"}},
				},
			},
			want: true,
		},
		{
			name:  "List Method - Primary Resource",
			field: &api.Field{Name: "parent"},
			method: &api.Method{
				Name: "ListInstances",
				InputType: &api.Message{
					Fields: []*api.Field{{Name: "parent"}},
				},
			},
			want: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := (&MethodAdapter{Method: test.method}).IsPrimaryResource(test.field)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetResource(t *testing.T) {
	instanceResource := &api.Resource{Type: "example.googleapis.com/Instance"}
	otherResource := &api.Resource{Type: "example.googleapis.com/Other"}

	for _, test := range []struct {
		name         string
		method       *api.Method
		resourceDefs []*api.Resource
		messages     []*api.Message
		want         *api.Resource
	}{
		{
			name: "Create Method - Resource in Message",
			method: &api.Method{
				Name: "CreateInstance",
				InputType: &api.Message{
					Fields: []*api.Field{
						{
							MessageType: &api.Message{
								Name:     "Instance",
								Resource: instanceResource,
							},
						},
					},
				},
			},
			resourceDefs: []*api.Resource{instanceResource},
			want:         instanceResource,
		},
		{
			name: "Get Method - Resource Reference",
			method: &api.Method{
				Name: "GetInstance",
				InputType: &api.Message{
					Fields: []*api.Field{
						api.NewTestField("name").WithResourceReference("example.googleapis.com/Instance"),
					},
				},
			},
			resourceDefs: []*api.Resource{instanceResource},
			want:         instanceResource,
		},
		{
			name: "List Method - Child Type Reference",
			method: &api.Method{
				Name: "ListInstances",
				InputType: &api.Message{
					Fields: []*api.Field{
						api.NewTestField("parent").WithChildTypeReference("example.googleapis.com/Instance"),
					},
				},
			},
			resourceDefs: []*api.Resource{instanceResource},
			want:         instanceResource,
		},
		{
			name: "Unknown Resource",
			method: &api.Method{
				Name: "Unknown",
				InputType: &api.Message{
					Fields: []*api.Field{{Name: "foo"}},
				},
			},
			resourceDefs: []*api.Resource{instanceResource},
			want:         nil,
		},
		{
			name: "Nil InputType",
			method: &api.Method{
				Name:      "NoInput",
				InputType: nil,
			},
			resourceDefs: []*api.Resource{instanceResource},
			want:         nil,
		},
		{
			name: "Resource on Message Directly",
			method: &api.Method{
				Name: "GetOther",
				InputType: &api.Message{
					Fields: []*api.Field{
						api.NewTestField("name").WithResourceReference("example.googleapis.com/Other"),
					},
				},
			},
			messages: []*api.Message{
				{
					Name:     "OtherMessage",
					Resource: otherResource,
				},
			},
			want: otherResource,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			model := &api.API{
				ResourceDefinitions: test.resourceDefs,
				Messages:            test.messages,
			}
			got := (&MethodAdapter{Method: test.method}).GetResource(model)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetPluralResourceName(t *testing.T) {
	instanceResource := &api.Resource{
		Type: "example.googleapis.com/Instance",
		Patterns: []api.ResourcePattern{
			parseResourcePattern("instances/{instance}"),
		},
	}

	for _, test := range []struct {
		name         string
		method       *api.Method
		resourceDefs []*api.Resource
		want         string
	}{
		{
			name: "Inferred from Pattern",
			method: &api.Method{
				Name: "ListInstances",
				InputType: &api.Message{
					Fields: []*api.Field{
						api.NewTestField("parent").WithChildTypeReference("example.googleapis.com/Instance"),
					},
				},
			},
			resourceDefs: []*api.Resource{instanceResource},
			want:         "instances",
		},
		{
			name: "Explicit Plural",
			method: &api.Method{
				Name: "ListBooks",
				InputType: &api.Message{
					Fields: []*api.Field{
						api.NewTestField("parent").WithChildTypeReference("example.googleapis.com/Book"),
					},
				},
			},
			resourceDefs: []*api.Resource{
				instanceResource,
				{
					Type:   "example.googleapis.com/Book",
					Plural: "books",
				},
			},
			want: "books",
		},
		{
			name: "Resource Not Found",
			method: &api.Method{
				Name: "ListUnknown",
				InputType: &api.Message{
					Fields: []*api.Field{
						api.NewTestField("parent").WithChildTypeReference("example.googleapis.com/Unknown"),
					},
				},
			},
			resourceDefs: []*api.Resource{instanceResource},
			want:         "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			model := &api.API{
				ResourceDefinitions: test.resourceDefs,
			}
			got := (&MethodAdapter{Method: test.method}).GetPluralResourceName(model)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetSingularResourceName(t *testing.T) {
	instanceResource := &api.Resource{
		Type: "example.googleapis.com/Instance",
		Patterns: []api.ResourcePattern{
			parseResourcePattern("instances/{instance}"),
		},
	}

	for _, test := range []struct {
		name         string
		method       *api.Method
		resourceDefs []*api.Resource
		want         string
	}{
		{
			name: "Inferred from Pattern",
			method: &api.Method{
				Name: "ListInstances",
				InputType: &api.Message{
					Fields: []*api.Field{
						api.NewTestField("parent").WithChildTypeReference("example.googleapis.com/Instance"),
					},
				},
			},
			resourceDefs: []*api.Resource{instanceResource},
			want:         "instance",
		},
		{
			name: "Explicit Singular",
			method: &api.Method{
				Name: "ListBooks",
				InputType: &api.Message{
					Fields: []*api.Field{
						api.NewTestField("parent").WithChildTypeReference("example.googleapis.com/Book"),
					},
				},
			},
			resourceDefs: []*api.Resource{
				instanceResource,
				{
					Type:     "example.googleapis.com/Book",
					Singular: "book",
				},
			},
			want: "book",
		},
		{
			name: "Resource Not Found",
			method: &api.Method{
				Name: "ListUnknown",
				InputType: &api.Message{
					Fields: []*api.Field{
						api.NewTestField("parent").WithChildTypeReference("example.googleapis.com/Unknown"),
					},
				},
			},
			resourceDefs: []*api.Resource{instanceResource},
			want:         "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			model := &api.API{
				ResourceDefinitions: test.resourceDefs,
			}
			got := (&MethodAdapter{Method: test.method}).GetSingularResourceName(model)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
