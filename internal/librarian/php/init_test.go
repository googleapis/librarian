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

package php

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestComponentName(t *testing.T) {
	for _, test := range []struct {
		name      string
		namespace string
		want      string
	}{
		{
			name:      "google cloud component",
			namespace: `Google\Cloud\SecretManager`,
			want:      "SecretManager",
		},
		{
			name:      "google ads",
			namespace: `Google\Ads\GoogleAds`,
			want:      "AdsGoogleAds",
		},
		{
			name:      "google shopping",
			namespace: `Google\Shopping\Merchant\Conversions`,
			want:      "ShoppingMerchantConversions",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := componentName(test.namespace)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
