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

package swift

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestExtractServiceDescription(t *testing.T) {
	for _, test := range []struct {
		name        string
		doc         string
		serviceName string
		want        string
	}{
		{
			name:        "empty documentation falls back to service name",
			doc:         "",
			serviceName: "SecretManagerService",
			want:        "Client for the SecretManagerService.",
		},
		{
			// SecretManagerService starts with a title line "Secret Manager Service"
			// without punctuation, followed by the actual description in the next paragraph.
			// A simple first-paragraph strategy would only extract the title, repeating
			// the client name without explaining what the service does.
			name: "SecretManagerService: skips title-only paragraph to extract description",
			doc: "Secret Manager Service\n\n" +
				"Manages secrets and operations using those secrets. Implements a REST\n" +
				"model with the following objects:\n\n" +
				"* [Secret][google.cloud.secretmanager.v1.Secret]\n" +
				"* [SecretVersion][google.cloud.secretmanager.v1.SecretVersion]",
			serviceName: "SecretManagerService",
			want:        "Manages secrets and operations using those secrets.",
		},
		{
			// EkmServiceClient similarly starts with a title line
			// "Google Cloud Key Management EKM Service" without terminal punctuation.
			// The second paragraph contains the actual description.
			name: "EkmServiceClient: skips title line to describe external key management",
			doc: "Google Cloud Key Management EKM Service\n\n" +
				"Manages external cryptographic keys and operations using those keys.\n" +
				"Implements a REST model with the following objects:\n" +
				"* [EkmConnection][google.cloud.kms.v1.EkmConnection]",
			serviceName: "EkmService",
			want:        "Manages external cryptographic keys and operations using those keys.",
		},
		{
			// AutoKeyClient's first paragraph contains a multi-sentence explanation
			// with implementation details and proto cross-reference links ([CryptoKeys][...],
			// [KeyHandle][...]). A simple full-paragraph extraction produces a bloated
			// multi-line block with broken reference link targets.
			// The extra complexity strips link targets and extracts only the first sentence.
			name: "AutoKeyClient: extracts first sentence and strips proto cross-reference links",
			doc: "Provides interfaces for using [Cloud KMS\n" +
				"Autokey](https://cloud.google.com/kms/help/autokey) to provision new\n" +
				"[CryptoKeys][google.cloud.kms.v1.CryptoKey], ready for Customer Managed\n" +
				"Encryption Key (CMEK) use, on-demand. To support certain client tooling, this\n" +
				"feature is modeled around a [KeyHandle][google.cloud.kms.v1.KeyHandle]\n" +
				"resource: creating a [KeyHandle][google.cloud.kms.v1.KeyHandle] in a resource\n" +
				"project and given location triggers Cloud KMS Autokey to provision a\n" +
				"[CryptoKey][google.cloud.kms.v1.CryptoKey] in the configured key project and\n" +
				"the same location.\n\n" +
				"Prior to use in a given resource project...",
			serviceName: "Autokey",
			want:        "Provides interfaces for using Cloud KMS Autokey to provision new CryptoKeys, ready for Customer Managed Encryption Key (CMEK) use, on-demand.",
		},
		{
			name:        "skips generic section header ending in colon",
			doc:         "API Overview:\n\nThe beyondcorp.googleapis.com service implements the Google Cloud BeyondCorp API.",
			serviceName: "AppConnectionsService",
			want:        "The beyondcorp.googleapis.com service implements the Google Cloud BeyondCorp API.",
		},
		{
			name:        "short title followed by description in same paragraph",
			doc:         "Google Batch Service.\nThe service manages user submitted batch jobs and allocates instances.",
			serviceName: "BatchService",
			want:        "Google Batch Service. The service manages user submitted batch jobs and allocates instances.",
		},
		{
			name:        "single paragraph with sentence",
			doc:         "Service to manage Security and Privacy Notifications.",
			serviceName: "NotificationsService",
			want:        "Service to manage Security and Privacy Notifications.",
		},
		{
			name:        "ensures terminal punctuation",
			doc:         "Google Cloud Key Management Service",
			serviceName: "KeyManagementService",
			want:        "Google Cloud Key Management Service.",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := extractServiceDescription(test.doc, test.serviceName)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
