// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package secrets

import (
	"context"
	"testing"

	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/gax-go/v2"
)

type mockClient struct {
	result string
}

func (c *mockClient) AccessSecretVersion(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error) {
	resp := &secretmanagerpb.AccessSecretVersionResponse{
		Payload: &secretmanagerpb.SecretPayload{
			Data: []byte(c.result),
		},
	}
	return resp, nil
}

func TestFetchSecrets(t *testing.T) {
	for _, test := range []struct {
		name       string
		project    string
		secretName string
		mockResult string
		want       string
	}{
		{"Basic", "some-project-id", "some-secret-name", "some-secret-value", "some-secret-value"},
		{"Empty response", "some-project-id", "some-secret-name", "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := &mockClient{
				result: test.mockResult,
			}
			got, err := Get(t.Context(), test.project, test.secretName, client)
			if err != nil {
				t.Errorf("unexpected error fetching secret: %s", err)
			}
			if diff := cmp.Diff(got, test.want); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
