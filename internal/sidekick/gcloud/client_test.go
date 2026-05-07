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

package gcloud

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestBuildClientCall(t *testing.T) {
	goClient := &goClientInfo{
		Alias:      "parallelstore",
		ClientPath: "cloud.google.com/go/parallelstore/apiv1",
		PbPath:     "cloud.google.com/go/parallelstore/apiv1/parallelstorepb",
	}

	for _, test := range []struct {
		name     string
		method   *api.Method
		goClient *goClientInfo
		hasPath  bool
		want     *ClientCall
	}{
		{
			name: "Get method",
			method: &api.Method{
				Name:      "GetInstance",
				InputType: &api.Message{Name: "GetInstanceRequest"},
			},
			goClient: goClient,
			hasPath:  true,
			want: &ClientCall{
				Method:      "GetInstance",
				NameField:   "Name",
				Package:     "parallelstore",
				RequestType: "parallelstorepb.GetInstanceRequest",
			},
		},
		{
			name: "List method",
			method: &api.Method{
				Name:      "ListInstances",
				InputType: &api.Message{Name: "ListInstancesRequest"},
			},
			goClient: goClient,
			hasPath:  true,
			want: &ClientCall{
				Method:      "ListInstances",
				NameField:   "Parent",
				Package:     "parallelstore",
				RequestType: "parallelstorepb.ListInstancesRequest",
				IsList:      true,
			},
		},
		{
			name: "Delete LRO method",
			method: &api.Method{
				Name:      "DeleteInstance",
				InputType: &api.Message{Name: "DeleteInstanceRequest"},
				IsLRO:     true,
			},
			goClient: goClient,
			hasPath:  true,
			want: &ClientCall{
				Method:      "DeleteInstance",
				NameField:   "Name",
				Package:     "parallelstore",
				RequestType: "parallelstorepb.DeleteInstanceRequest",
				IsDelete:    true,
				IsLRO:       true,
			},
		},
		{
			name: "Delete non-LRO method",
			method: &api.Method{
				Name:      "DeleteOperation",
				InputType: &api.Message{Name: "DeleteOperationRequest"},
			},
			goClient: goClient,
			hasPath:  true,
			want: &ClientCall{
				Method:      "DeleteOperation",
				NameField:   "Name",
				Package:     "parallelstore",
				RequestType: "parallelstorepb.DeleteOperationRequest",
				IsDelete:    true,
			},
		},
		{
			name: "nil goClient",
			method: &api.Method{
				Name:      "GetInstance",
				InputType: &api.Message{Name: "GetInstanceRequest"},
			},
			goClient: nil,
			hasPath:  true,
			want:     nil,
		},
		{
			name: "no path",
			method: &api.Method{
				Name:      "GetInstance",
				InputType: &api.Message{Name: "GetInstanceRequest"},
			},
			goClient: goClient,
			hasPath:  false,
			want:     nil,
		},
		{
			name: "nil InputType",
			method: &api.Method{
				Name: "GetInstance",
			},
			goClient: goClient,
			hasPath:  true,
			want:     nil,
		},
		{
			name: "not get list or delete",
			method: &api.Method{
				Name:      "ImportData",
				InputType: &api.Message{Name: "ImportDataRequest"},
			},
			goClient: goClient,
			hasPath:  true,
			want:     nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := buildClientCall(test.method, test.goClient, test.hasPath)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
