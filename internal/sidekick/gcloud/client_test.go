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

	// createModel is a minimal model with an Instance resource definition
	// whose body has a couple of scalar fields plus one of every kind that
	// must be skipped with a TODO.
	createInstance := &api.Message{
		Name: "Instance",
		ID:   ".google.cloud.parallelstore.v1.Instance",
		Fields: []*api.Field{
			{Name: "name", Typez: api.TypezString, Behavior: []api.FieldBehavior{api.FieldBehaviorIdentifier}},
			{Name: "description", Typez: api.TypezString},
			{Name: "state", Typez: api.TypezEnum},
			{Name: "labels", Typez: api.TypezMessage, Map: true},
			{Name: "capacity_gib", Typez: api.TypezInt64, Behavior: []api.FieldBehavior{api.FieldBehaviorRequired}},
			{Name: "access_points", Typez: api.TypezString, Repeated: true, Behavior: []api.FieldBehavior{api.FieldBehaviorOutputOnly}},
			{Name: "create_time", Typez: api.TypezMessage, Behavior: []api.FieldBehavior{api.FieldBehaviorOutputOnly}},
		},
	}
	createInstance.Resource = &api.Resource{
		Type:     "parallelstore.googleapis.com/Instance",
		Singular: "instance",
		Self:     createInstance,
	}
	createModel := &api.API{
		Name:        "parallelstore",
		PackageName: "google.cloud.parallelstore.v1",
		Messages:    []*api.Message{createInstance},
	}
	createMethod := &api.Method{
		Name:                "CreateInstance",
		IsAIPStandardCreate: true,
		IsLRO:               true,
		InputType: &api.Message{
			Name: "CreateInstanceRequest",
			Fields: []*api.Field{
				{Name: "parent", Typez: api.TypezString},
				{Name: "instance_id", Typez: api.TypezString, Behavior: []api.FieldBehavior{api.FieldBehaviorRequired}},
				{Name: "instance", Typez: api.TypezMessage, TypezID: createInstance.ID, MessageType: createInstance, Behavior: []api.FieldBehavior{api.FieldBehaviorRequired}},
			},
		},
	}

	for _, test := range []struct {
		name      string
		method    *api.Method
		model     *api.API
		goClient  *goClientInfo
		hasPath   bool
		want      *ClientCall
		wantFlags []Flag
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
				Paged:       true,
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
			name:     "Create method",
			method:   createMethod,
			model:    createModel,
			goClient: goClient,
			hasPath:  true,
			want: &ClientCall{
				IsCreate:    true,
				IsLRO:       true,
				Method:      "CreateInstance",
				NameField:   "Parent",
				Package:     "parallelstore",
				RequestType: "parallelstorepb.CreateInstanceRequest",
				IDField:     "InstanceId",
				IDFlag:      "instance-id",
				BodyField:   "Instance",
				BodyType:    "parallelstorepb.Instance",
				BodyAssignments: []BodyAssignment{
					{Name: "Description", Flag: "description", Kind: "String"},
					{Name: "CapacityGib", Flag: "capacity-gib", Kind: "Int64"},
				},
				BodySkippedFields: []string{
					`enum field "state"`,
					`map field "labels"`,
				},
			},
			wantFlags: []Flag{
				{Name: "instance-id", Kind: "String", Required: true, Usage: "The instance id."},
				{Name: "description", Kind: "String", Usage: "The description."},
				{Name: "capacity-gib", Kind: "Int64", Required: true, Usage: "The capacity gib."},
			},
		},
		{
			name: "ImportData LRO method",
			method: &api.Method{
				Name:      "ImportData",
				InputType: &api.Message{Name: "ImportDataRequest"},
				IsLRO:     true,
			},
			goClient: goClient,
			hasPath:  true,
			want:     nil,
		},
		{
			name: "ExportData LRO method",
			method: &api.Method{
				Name:      "ExportData",
				InputType: &api.Message{Name: "ExportDataRequest"},
				IsLRO:     true,
			},
			goClient: goClient,
			hasPath:  true,
			want:     nil,
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
			name: "unknown method",
			method: &api.Method{
				Name:      "DoSomethingCustom",
				InputType: &api.Message{Name: "DoSomethingCustomRequest"},
			},
			goClient: goClient,
			hasPath:  true,
			want:     nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotCall, gotFlags := buildClientCall(test.method, test.model, test.goClient, test.hasPath)
			if diff := cmp.Diff(test.want, gotCall); diff != "" {
				t.Errorf("call mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantFlags, gotFlags); diff != "" {
				t.Errorf("flags mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
