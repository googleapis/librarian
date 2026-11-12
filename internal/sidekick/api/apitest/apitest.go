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

// Package apitest provides helper functions for testing the api package.
package apitest

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

// IgnoreFieldNames returns a [cmp.Option] that ignores the named unexported
// fields on [api.Field].
func IgnoreFieldNames(names ...string) cmp.Option {
	set := make(map[string]bool, len(names))
	for _, n := range names {
		set["."+n] = true
	}
	fieldType := reflect.TypeFor[api.Field]()
	return cmp.FilterPath(func(p cmp.Path) bool {
		if len(p) < 2 {
			return false
		}
		if p.Index(-2).Type() != fieldType {
			return false
		}
		return set[p.Last().String()]
	}, cmp.Ignore())
}

// AllowUnexported returns a [cmp.Option] that permits comparing unexported
// fields on all read-only [api] AST and metadata structs.
func AllowUnexported() cmp.Option {
	return cmp.AllowUnexported(
		api.DiscoveryLro{},
		api.Enum{},
		api.EnumValue{},
		api.Field{},
		api.Map{},
		api.MethodSignature{},
		api.OneOf{},
		api.OperationInfo{},
		api.PathBinding{},
		api.PathInfo{},
		api.PathSegment{},
		api.PathTemplate{},
		api.PathVariable{},
		api.Resource{},
		api.ResourceNamePattern{},
		api.ResourceNameSegment{},
		api.ResourceReference{},
		api.RoutingInfo{},
		api.RoutingInfoCombo{},
		api.RoutingInfoComboItem{},
		api.RoutingInfoVariant{},
		api.RoutingPathSpec{},
		api.SampleValue{},
		api.TargetResource{},
	)
}

// CmpOptions returns standard comparison options for api types, including
// AllowUnexported() and EquateEmpty().
func CmpOptions() cmp.Option {
	return cmp.Options{
		AllowUnexported(),
		cmpopts.EquateEmpty(),
	}
}

// CheckMessage compares two `Message` instances ignoring the order of fields, and oneofs and ignoring child messages.
func CheckMessage(t *testing.T, got *api.Message, want *api.Message) {
	t.Helper()
	// Checking Parent, Messages, Fields, and OneOfs requires special handling.
	if diff := cmp.Diff(want, got,
		AllowUnexported(),
		cmpopts.IgnoreFields(api.Message{}, "Fields", "OneOfs", "Parent", "Messages", "Enums", "Resource"),
		IgnoreFieldNames("parent"),
	); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	less := func(a, b *api.Field) bool { return a.Name() < b.Name() }
	if diff := cmp.Diff(want.Fields, got.Fields,
		AllowUnexported(),
		cmpopts.SortSlices(less),
		IgnoreFieldNames("parent"),
	); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	// Ignore parent because types are cyclic
	if diff := cmp.Diff(want.OneOfs, got.OneOfs,
		AllowUnexported(),
		cmpopts.SortSlices(less),
		IgnoreFieldNames("parent"),
	); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

// CheckEnum compares two `Enum` instances ignoring the enum value order.
func CheckEnum(t *testing.T, got api.Enum, want api.Enum) {
	t.Helper()
	ignoreEnumFields := cmp.FilterPath(func(p cmp.Path) bool {
		switch p.Last().String() {
		case ".values", ".uniqueNumberValues", ".parent":
			return true
		}
		return false
	}, cmp.Ignore())
	if diff := cmp.Diff(want, got,
		AllowUnexported(),
		ignoreEnumFields,
	); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	less := func(a, b *api.EnumValue) bool { return a.Name() < b.Name() }
	if diff := cmp.Diff(want.Values(), got.Values(),
		AllowUnexported(),
		cmpopts.SortSlices(less),
		cmp.FilterPath(func(p cmp.Path) bool { return p.Last().String() == ".parent" }, cmp.Ignore()),
	); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

// CheckService compares two `Service` instances ignoring method order.
func CheckService(t *testing.T, got *api.Service, want *api.Service) {
	t.Helper()
	if diff := cmp.Diff(want, got, AllowUnexported(), cmpopts.IgnoreFields(api.Service{}, "Methods")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	less := func(a, b *api.Method) bool { return a.Name < b.Name }
	if diff := cmp.Diff(want.Methods, got.Methods,
		AllowUnexported(),
		cmpopts.SortSlices(less),
		IgnoreFieldNames("parent"),
		cmpopts.IgnoreFields(api.Method{}, "Service", "Model"),
	); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

// CheckMethod finds a `Method` in a `Service` and compares the values.
func CheckMethod(t *testing.T, service *api.Service, name string, want *api.Method) {
	t.Helper()
	findMethod := func(name string) (*api.Method, bool) {
		for _, method := range service.Methods {
			if method.Name == name {
				return method, true
			}
		}
		return nil, false
	}
	got, ok := findMethod(name)
	if !ok {
		t.Errorf("missing method %s", name)
	}
	if diff := cmp.Diff(want, got,
		AllowUnexported(),
		IgnoreFieldNames("parent"),
		cmpopts.IgnoreFields(api.Method{}, "Service", "Model"),
	); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

// CheckMap compares two `Map` instances.
func CheckMap(t *testing.T, got *api.Map, want *api.Map) {
	t.Helper()
	if diff := cmp.Diff(want, got, AllowUnexported(), IgnoreFieldNames("parent")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
