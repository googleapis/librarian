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

// Package secrets provides the interface for interacting with Secret Manager
package secrets

import (
	"context"
	"fmt"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

// Fetch the latest version of a secret as a string. This method assumes the
// secret payload is a UTF-8 string.
func Get(ctx context.Context, project string, secretName string) (string, error) {
	secretsClient, err := secretmanager.NewClient(ctx)
	if err != nil {
		return "", err
	}
	defer secretsClient.Close()
	request := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", project, secretName),
	}
	secret, err := secretsClient.AccessSecretVersion(ctx, request)
	if err != nil {
		return "", err
	}
	// We assume the payload is valid UTF-8.
	value := string(secret.Payload.Data[:])
	return value, nil
}
