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
	"testing"
)

func TestGetVerb(t *testing.T) {
	for _, test := range []struct {
		name       string
		methodName string
		want       string
		wantErr    bool
	}{
		{"Get", "GetInstance", "describe", false},
		{"List", "ListInstances", "list", false},
		{"Create", "CreateInstance", "create", false},
		{"Update", "UpdateInstance", "update", false},
		{"Delete", "DeleteInstance", "delete", false},
		{"Custom", "DetachDisk", "detach_disk", false},
		{"Empty", "", "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := GetVerb(test.methodName)
			if (err != nil) != test.wantErr {
				t.Errorf("GetVerb(%q) error = %v, wantErr %v", test.methodName, err, test.wantErr)
				return
			}
			if got != test.want {
				t.Errorf("GetVerb(%q) = %q, want %q", test.methodName, got, test.want)
			}
		})
	}
}

func TestIsCreate(t *testing.T) {
	for _, test := range []struct {
		name       string
		methodName string
		want       bool
	}{
		{"True", "CreateInstance", true},
		{"False", "GetInstance", false},
		{"Empty", "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := IsCreate(test.methodName); got != test.want {
				t.Errorf("IsCreate(%q) = %v, want %v", test.methodName, got, test.want)
			}
		})
	}
}

func TestIsGet(t *testing.T) {
	for _, test := range []struct {
		name       string
		methodName string
		want       bool
	}{
		{"True", "GetInstance", true},
		{"False", "CreateInstance", false},
		{"Empty", "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := IsGet(test.methodName); got != test.want {
				t.Errorf("IsGet(%q) = %v, want %v", test.methodName, got, test.want)
			}
		})
	}
}

func TestIsList(t *testing.T) {
	for _, test := range []struct {
		name       string
		methodName string
		want       bool
	}{
		{"True", "ListInstances", true},
		{"False", "GetInstance", false},
		{"Empty", "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := IsList(test.methodName); got != test.want {
				t.Errorf("IsList(%q) = %v, want %v", test.methodName, got, test.want)
			}
		})
	}
}

func TestIsUpdate(t *testing.T) {
	for _, test := range []struct {
		name       string
		methodName string
		want       bool
	}{
		{"True", "UpdateInstance", true},
		{"False", "GetInstance", false},
		{"Empty", "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := IsUpdate(test.methodName); got != test.want {
				t.Errorf("IsUpdate(%q) = %v, want %v", test.methodName, got, test.want)
			}
		})
	}
}

func TestIsDelete(t *testing.T) {
	for _, test := range []struct {
		name       string
		methodName string
		want       bool
	}{
		{"True", "DeleteInstance", true},
		{"False", "GetInstance", false},
		{"Empty", "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := IsDelete(test.methodName); got != test.want {
				t.Errorf("IsDelete(%q) = %v, want %v", test.methodName, got, test.want)
			}
		})
	}
}