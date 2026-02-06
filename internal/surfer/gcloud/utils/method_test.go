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

package utils

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
		{"Verb Match", &api.Method{Name: "CreateInstance", PathInfo: &api.PathInfo{Bindings: []*api.PathBinding{{Verb: "POST"}}}}, true},
		{"Verb Mismatch", &api.Method{Name: "CreateInstance", PathInfo: &api.PathInfo{Bindings: []*api.PathBinding{{Verb: "GET"}}}}, false},
	} {
		test := test
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
		{"Verb Match", &api.Method{Name: "GetInstance", PathInfo: &api.PathInfo{Bindings: []*api.PathBinding{{Verb: "GET"}}}}, true},
		{"Verb Mismatch", &api.Method{Name: "GetInstance", PathInfo: &api.PathInfo{Bindings: []*api.PathBinding{{Verb: "POST"}}}}, false},
	} {
		test := test
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
		{"Verb Match", &api.Method{Name: "ListInstances", PathInfo: &api.PathInfo{Bindings: []*api.PathBinding{{Verb: "GET"}}}}, true},
		{"Verb Mismatch", &api.Method{Name: "ListInstances", PathInfo: &api.PathInfo{Bindings: []*api.PathBinding{{Verb: "POST"}}}}, false},
	} {
		test := test
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
		{"Verb Match PATCH", &api.Method{Name: "UpdateInstance", PathInfo: &api.PathInfo{Bindings: []*api.PathBinding{{Verb: "PATCH"}}}}, true},
		{"Verb Match PUT", &api.Method{Name: "UpdateInstance", PathInfo: &api.PathInfo{Bindings: []*api.PathBinding{{Verb: "PUT"}}}}, true},
		{"Verb Mismatch", &api.Method{Name: "UpdateInstance", PathInfo: &api.PathInfo{Bindings: []*api.PathBinding{{Verb: "GET"}}}}, false},
	} {
		test := test
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
		{"Verb Match", &api.Method{Name: "DeleteInstance", PathInfo: &api.PathInfo{Bindings: []*api.PathBinding{{Verb: "DELETE"}}}}, true},
		{"Verb Mismatch", &api.Method{Name: "DeleteInstance", PathInfo: &api.PathInfo{Bindings: []*api.PathBinding{{Verb: "GET"}}}}, false},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := IsDelete(test.method)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsCustomMethod(t *testing.T) {
	for _, test := range []struct {
		name   string
		method *api.Method
		want   bool
	}{
		{"Standard Get", &api.Method{Name: "GetInstance", PathInfo: &api.PathInfo{Bindings: []*api.PathBinding{{Verb: "GET"}}}}, false},
		{"Standard List", &api.Method{Name: "ListInstances", PathInfo: &api.PathInfo{Bindings: []*api.PathBinding{{Verb: "GET"}}}}, false},
		{"Standard Create", &api.Method{Name: "CreateInstance", PathInfo: &api.PathInfo{Bindings: []*api.PathBinding{{Verb: "POST"}}}}, false},
		{"Standard Update", &api.Method{Name: "UpdateInstance", PathInfo: &api.PathInfo{Bindings: []*api.PathBinding{{Verb: "PATCH"}}}}, false},
		{"Standard Delete", &api.Method{Name: "DeleteInstance", PathInfo: &api.PathInfo{Bindings: []*api.PathBinding{{Verb: "DELETE"}}}}, false},
		{"Custom Method", &api.Method{Name: "ExportData", PathInfo: &api.PathInfo{Bindings: []*api.PathBinding{{Verb: "POST"}}}}, true},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := IsCustomMethod(test.method); got != test.want {
				t.Errorf("mismatch (-want +got):\n%s", cmp.Diff(test.want, got))
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
		{"Custom Verb in Path", &api.Method{
			Name: "ExportData",
			PathInfo: &api.PathInfo{
				Bindings: []*api.PathBinding{
					{
						PathTemplate: &api.PathTemplate{
							Verb: &v,
						},
					},
				},
			},
		}, "export_data"},
		{"Fallback to Name", &api.Method{Name: "ExportData"}, "export_data"},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := GetCommandName(test.method)
			if err != nil {
				t.Fatalf("GetCommandName() error = %v", err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
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
			got := FindResourceMessage(test.outputType)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("FindResourceMessage mismatch (-want +got):\n%s", diff)
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
		test := test
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
		{"Standard Get", &api.Method{Name: "GetInstance", PathInfo: &api.PathInfo{Bindings: []*api.PathBinding{{Verb: "GET"}}}}, true},
		{"Standard List", &api.Method{Name: "ListInstances", PathInfo: &api.PathInfo{Bindings: []*api.PathBinding{{Verb: "GET"}}}}, false},
		{"Custom Resource", &api.Method{
			Name: "CustomInstance",
			PathInfo: &api.PathInfo{Bindings: []*api.PathBinding{{
				PathTemplate: &api.PathTemplate{Segments: []api.PathSegment{*api.NewPathSegment().WithVariable(api.NewPathVariable("instance"))}},
			}}},
		}, true},
		{"Custom Collection", &api.Method{
			Name: "CustomCollection",
			PathInfo: &api.PathInfo{Bindings: []*api.PathBinding{{
				PathTemplate: &api.PathTemplate{Segments: []api.PathSegment{*api.NewPathSegment().WithLiteral("instances")}},
			}}},
		}, false},
	} {
		test := test
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
		{"Standard Get", &api.Method{Name: "GetInstance", PathInfo: &api.PathInfo{Bindings: []*api.PathBinding{{Verb: "GET"}}}}, false},
		{"Standard List", &api.Method{Name: "ListInstances", PathInfo: &api.PathInfo{Bindings: []*api.PathBinding{{Verb: "GET"}}}}, true},
		{"Custom Resource", &api.Method{
			Name: "CustomInstance",
			PathInfo: &api.PathInfo{Bindings: []*api.PathBinding{{
				PathTemplate: &api.PathTemplate{Segments: []api.PathSegment{*api.NewPathSegment().WithVariable(api.NewPathVariable("instance"))}},
			}}},
		}, false},
		{"Custom Collection", &api.Method{
			Name: "CustomCollection",
			PathInfo: &api.PathInfo{Bindings: []*api.PathBinding{{
				PathTemplate: &api.PathTemplate{Segments: []api.PathSegment{*api.NewPathSegment().WithLiteral("instances")}},
			}}},
		}, true},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := IsCollectionMethod(test.method); got != test.want {
				t.Errorf("mismatch (-want +got):\n%s", cmp.Diff(test.want, got))
			}
		})
	}
}
