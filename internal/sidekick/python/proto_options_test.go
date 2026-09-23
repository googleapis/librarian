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

package python

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseProtoServiceBlock_OAuthScopesAndExtendedOperations(t *testing.T) {
	proto := `
syntax = "proto3";

package google.cloud.compute.v1;

service OtherService {
  option (google.api.oauth_scopes) = "https://www.googleapis.com/auth/other";
  rpc OtherRpc(OtherReq) returns (OtherResp) {
    option (google.cloud.operation_service) = "OtherOperations";
  }
}

service Instances {
  option (google.api.default_host) =
    "compute.googleapis.com";

  option (google.api.oauth_scopes) =
    "https://www.googleapis.com/auth/compute,"
    "https://www.googleapis.com/auth/cloud-platform";

  rpc AddAccessConfig(AddAccessConfigRequest) returns (Operation) {
    option (google.api.http) = {
      post: "/compute/v1/projects/{project}/zones/{zone}/instances/{instance}/addAccessConfig"
    };
    option (google.cloud.operation_service) = "ZoneOperations";
  }

  rpc AggregatedList(AggregatedListRequest) returns (InstancesScopedList) {
    option (google.api.http) = {
      get: "/compute/v1/projects/{project}/aggregated/instances"
    };
  }

  rpc Delete(DeleteInstanceRequest) returns (Operation) {
    option (google.cloud.operation_service) = "ZoneOperations";
  }

  rpc EmptyMethod(EmptyReq) returns (EmptyResp);
}
`

	opts := &protoServiceOptions{
		MethodOperationServices: make(map[string]string),
	}
	parseProtoServiceBlock(proto, "Instances", opts)

	wantScopes := []string{
		"https://www.googleapis.com/auth/compute",
		"https://www.googleapis.com/auth/cloud-platform",
	}
	if diff := cmp.Diff(wantScopes, opts.OAuthScopes); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantOps := map[string]string{
		"AddAccessConfig": "ZoneOperations",
		"Delete":          "ZoneOperations",
	}
	if diff := cmp.Diff(wantOps, opts.MethodOperationServices); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantExtSvcs := []*extendedOperationService{
		{ServiceNameSnake: "zone_operations", ServiceNamePascal: "ZoneOperations"},
	}
	if diff := cmp.Diff(wantExtSvcs, opts.ExtendedOperationsServices); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestParseProtoServiceBlock_NotFound(t *testing.T) {
	opts := &protoServiceOptions{
		MethodOperationServices: make(map[string]string),
	}
	parseProtoServiceBlock("syntax = \"proto3\";", "MissingService", opts)

	if len(opts.OAuthScopes) != 0 {
		t.Errorf("expected empty OAuthScopes, got %v", opts.OAuthScopes)
	}
	if len(opts.MethodOperationServices) != 0 {
		t.Errorf("expected empty MethodOperationServices, got %v", opts.MethodOperationServices)
	}
}
