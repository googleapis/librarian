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

package api

import "strings"

// Each element of the model (services, messages, enums) has a series of
// "scopes" associated with it. These are the relative names for symbols
// in the context of the element.
//
// This is intended for discovery of relative and absolute cross-reference links
// in the documentation.
//
// For example, with a proto specification like:
//
// ```proto
// package .test.v1;
//
// message M {
//   message Child {
//     string f1 = 1;
//   }
//   string f1 = 1;
//   Child f2 = 2;
// }
// ```
//
// In the context of `Child` we may say `[f1][]` and that is a cross-reference
// link to `.test.v1.M.Child.f1`.  We may also refer to the same field as
// `[Child.f1][]` or `[M.Child.f1][]` or even `[.test.v1.M.Child.f1]][]`.
//
// In the context of `M` when we say `[f1][]` that refers to
// `.test.v1.M.f1`.

// Scopes returns the scopes for a service.
func (x *Service) Scopes() []string {
	return []string{strings.TrimPrefix(x.ID, "."), x.Package}
}

// Scopes returns the scopes for a message.
func (x *Message) Scopes() []string {
	localScope := strings.TrimPrefix(x.ID, ".")
	if x.Parent == nil {
		return []string{localScope, x.Package} // simplify some test set-up
	}
	return append([]string{localScope}, x.Parent.Scopes()...)
}

// Scopes returns the scopes for an enum.
func (x *Enum) Scopes() []string {
	localScope := strings.TrimPrefix(x.ID, ".")
	if x.Parent == nil {
		return []string{localScope, x.Package} // simplify some test set-up
	}
	return append([]string{localScope}, x.Parent.Scopes()...)
}

// Scopes returns the scopes for an enum value.
func (x *EnumValue) Scopes() []string {
	if x.Parent != nil {
		return x.Parent.Scopes()
	}
	return fallbackScopes(x.ID)
}

// Scopes returns the scopes for a field.
func (x *Field) Scopes() []string {
	if x.Parent != nil {
		return x.Parent.Scopes()
	}
	return fallbackScopes(x.ID)
}

// Scopes returns the scopes for a method.
func (x *Method) Scopes() []string {
	if x.Service != nil {
		return x.Service.Scopes()
	}
	return fallbackScopes(x.ID)
}

// Scopes returns the scopes for a oneof.
func (x *OneOf) Scopes() []string {
	if len(x.Fields) > 0 {
		return x.Fields[0].Scopes()
	}
	return fallbackScopes(x.ID)
}

// A fallback so we can be lazy in test set-up.
func fallbackScopes(id string) []string {
	parts := strings.Split(strings.TrimPrefix(id, "."), ".")
	if len(parts) <= 1 {
		return []string{}
	}
	parts = parts[:len(parts)-1]
	res := make([]string, 0, len(parts))
	for i := len(parts); i > 0; i-- {
		res = append(res, strings.Join(parts[:i], "."))
	}
	return res
}
