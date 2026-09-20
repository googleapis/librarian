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
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

// CheckMessage compares two `Message` instances ignoring the order of fields, and oneofs and ignoring child messages.
func CheckMessage(t *testing.T, got *api.Message, want *api.Message) {
	t.Helper()
	ignored := []string{"Fields", "OneOfs", "Parent", "Messages", "Enums", "Resource"}
	if want.SourceLocation == nil {
		ignored = append(ignored, "SourceLocation")
	}

	hasFieldNumber := slices.ContainsFunc(want.Fields, func(f *api.Field) bool { return f.Number != 0 }) ||
		slices.ContainsFunc(want.OneOfs, func(o *api.OneOf) bool {
			return slices.ContainsFunc(o.Fields, func(f *api.Field) bool { return f.Number != 0 })
		}) ||
		(want.Pagination != nil && ((want.Pagination.NextPageToken != nil && want.Pagination.NextPageToken.Number != 0) ||
			(want.Pagination.PageableItem != nil && want.Pagination.PageableItem.Number != 0)))

	hasFieldSourceLoc := slices.ContainsFunc(want.Fields, func(f *api.Field) bool { return f.SourceLocation != nil }) ||
		slices.ContainsFunc(want.OneOfs, func(o *api.OneOf) bool {
			return slices.ContainsFunc(o.Fields, func(f *api.Field) bool { return f.SourceLocation != nil })
		}) ||
		(want.Pagination != nil && ((want.Pagination.NextPageToken != nil && want.Pagination.NextPageToken.SourceLocation != nil) ||
			(want.Pagination.PageableItem != nil && want.Pagination.PageableItem.SourceLocation != nil)))

	fieldOpts := fieldIgnoreOpts(hasFieldNumber, hasFieldSourceLoc)

	msgOpts := append([]cmp.Option{cmpopts.IgnoreFields(api.Message{}, ignored...)}, fieldOpts...)
	if diff := cmp.Diff(want, got, msgOpts...); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	lessField := func(a, b *api.Field) bool { return a.Name < b.Name }
	sliceFieldOpts := append([]cmp.Option{cmpopts.SortSlices(lessField)}, fieldOpts...)
	if diff := cmp.Diff(want.Fields, got.Fields, sliceFieldOpts...); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	lessOneOf := func(a, b *api.OneOf) bool { return a.Name < b.Name }
	oneofOpts := []cmp.Option{cmpopts.SortSlices(lessOneOf)}
	if !slices.ContainsFunc(want.OneOfs, func(o *api.OneOf) bool { return o.SourceLocation != nil }) {
		oneofOpts = append(oneofOpts, cmpopts.IgnoreFields(api.OneOf{}, "SourceLocation"))
	}
	oneofOpts = append(oneofOpts, fieldOpts...)
	if diff := cmp.Diff(want.OneOfs, got.OneOfs, oneofOpts...); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

// CheckEnum compares two `Enum` instances ignoring the enum value order.
func CheckEnum(t *testing.T, got api.Enum, want api.Enum) {
	t.Helper()
	ignored := []string{"Values", "UniqueNumberValues", "Parent"}
	if want.SourceLocation == nil {
		ignored = append(ignored, "SourceLocation")
	}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(api.Enum{}, ignored...)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	less := func(a, b *api.EnumValue) bool { return a.Name < b.Name }
	valueIgnored := []string{"Parent"}
	if !slices.ContainsFunc(want.Values, func(v *api.EnumValue) bool { return v.SourceLocation != nil }) {
		valueIgnored = append(valueIgnored, "SourceLocation")
	}
	if diff := cmp.Diff(want.Values, got.Values, cmpopts.SortSlices(less), cmpopts.IgnoreFields(api.EnumValue{}, valueIgnored...)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

// CheckService compares two `Service` instances ignoring method order.
func CheckService(t *testing.T, got *api.Service, want *api.Service) {
	t.Helper()
	ignored := []string{"Methods"}
	if want.SourceLocation == nil {
		ignored = append(ignored, "SourceLocation")
	}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(api.Service{}, ignored...)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	less := func(a, b *api.Method) bool { return a.Name < b.Name }
	var methodOpts []cmp.Option
	methodOpts = append(methodOpts, cmpopts.SortSlices(less))
	if !slices.ContainsFunc(want.Methods, func(m *api.Method) bool { return m.SourceLocation != nil }) {
		methodOpts = append(methodOpts, cmpopts.IgnoreFields(api.Method{}, "SourceLocation"))
	}
	hasMethodFieldNumber := slices.ContainsFunc(want.Methods, func(m *api.Method) bool {
		return (m.Pagination != nil && m.Pagination.Number != 0) ||
			slices.ContainsFunc(m.AutoPopulated, func(f *api.Field) bool { return f.Number != 0 })
	})
	hasMethodFieldSourceLoc := slices.ContainsFunc(want.Methods, func(m *api.Method) bool {
		return (m.Pagination != nil && m.Pagination.SourceLocation != nil) ||
			slices.ContainsFunc(m.AutoPopulated, func(f *api.Field) bool { return f.SourceLocation != nil })
	})
	methodOpts = append(methodOpts, fieldIgnoreOpts(hasMethodFieldNumber, hasMethodFieldSourceLoc)...)
	if diff := cmp.Diff(want.Methods, got.Methods, methodOpts...); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

// CheckMethod finds a `Method` in a `Service` and compares the values.
func CheckMethod(t *testing.T, service *api.Service, name string, want *api.Method) {
	t.Helper()
	idx := slices.IndexFunc(service.Methods, func(m *api.Method) bool { return m.Name == name })
	if idx == -1 {
		t.Fatalf("service %s missing method %s", service.ID, name)
	}
	got := service.Methods[idx]
	var opts []cmp.Option
	if want.SourceLocation == nil {
		opts = append(opts, cmpopts.IgnoreFields(api.Method{}, "SourceLocation"))
	}
	hasFieldNum := (want.Pagination != nil && want.Pagination.Number != 0) ||
		slices.ContainsFunc(want.AutoPopulated, func(f *api.Field) bool { return f.Number != 0 })
	hasFieldLoc := (want.Pagination != nil && want.Pagination.SourceLocation != nil) ||
		slices.ContainsFunc(want.AutoPopulated, func(f *api.Field) bool { return f.SourceLocation != nil })
	opts = append(opts, fieldIgnoreOpts(hasFieldNum, hasFieldLoc)...)
	if diff := cmp.Diff(want, got, opts...); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func fieldIgnoreOpts(hasNumber, hasSourceLocation bool) []cmp.Option {
	var ignored []string
	if !hasNumber {
		ignored = append(ignored, "Number")
	}
	if !hasSourceLocation {
		ignored = append(ignored, "SourceLocation")
	}
	if len(ignored) > 0 {
		return []cmp.Option{cmpopts.IgnoreFields(api.Field{}, ignored...)}
	}
	return nil
}
