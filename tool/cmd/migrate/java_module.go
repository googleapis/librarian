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

package main

var (
	apiShortnameOverrides = map[string]string{
		"beyondcorp-appconnections":          "beyondcorp-appconnections",
		"beyondcorp-appconnectors":           "beyondcorp-appconnectors",
		"beyondcorp-appgateways":             "beyondcorp-appgateways",
		"beyondcorp-clientconnectorservices": "beyondcorp-clientconnectorservices",
		"beyondcorp-clientgateways":          "beyondcorp-clientgateways",
		"dialogflow-cx":                      "dialogflow-cx",
		"distributedcloudedge":               "distributedcloudedge",
		"gke-backup":                         "gke-backup",
		"apigee-registry":                    "apigee-registry",
	}

	excludedSamplesLibraries = map[string]bool{
		"bigquerystorage": true,
		"datastore":       true,
		"logging":         true,
		"storage":         true,
		"spanner":         true,
	}

	keepOverride = map[string][]string{
		"translate": {
			"google-cloud-translate/src/main/java/com/google/cloud/translate/Detection.java",
			"google-cloud-translate/src/main/java/com/google/cloud/translate/Language.java",
			"google-cloud-translate/src/main/java/com/google/cloud/translate/Option.java",
			"google-cloud-translate/src/main/java/com/google/cloud/translate/Translate.java",
			"google-cloud-translate/src/main/java/com/google/cloud/translate/TranslateException.java",
			"google-cloud-translate/src/main/java/com/google/cloud/translate/TranslateFactory.java",
			"google-cloud-translate/src/main/java/com/google/cloud/translate/TranslateImpl.java",
			"google-cloud-translate/src/main/java/com/google/cloud/translate/TranslateOptions.java",
			"google-cloud-translate/src/main/java/com/google/cloud/translate/Translation.java",
			"google-cloud-translate/src/main/java/com/google/cloud/translate/package-info.java",
			"google-cloud-translate/src/main/java/com/google/cloud/translate/spi/TranslateRpcFactory.java",
			"google-cloud-translate/src/main/java/com/google/cloud/translate/spi/v2/HttpTranslateRpc.java",
			"google-cloud-translate/src/main/java/com/google/cloud/translate/spi/v2/TranslateRpc.java",
			"google-cloud-translate/src/main/java/com/google/cloud/translate/testing/RemoteTranslateHelper.java",
			"google-cloud-translate/src/main/java/com/google/cloud/translate/testing/package-info.java",
			"google-cloud-translate/src/test/java/com/google/cloud/translate/DetectionTest.java",
			"google-cloud-translate/src/test/java/com/google/cloud/translate/LanguageTest.java",
			"google-cloud-translate/src/test/java/com/google/cloud/translate/OptionTest.java",
			"google-cloud-translate/src/test/java/com/google/cloud/translate/SerializationTest.java",
			"google-cloud-translate/src/test/java/com/google/cloud/translate/TranslateExceptionTest.java",
			"google-cloud-translate/src/test/java/com/google/cloud/translate/TranslateImplTest.java",
			"google-cloud-translate/src/test/java/com/google/cloud/translate/TranslateOptionsTest.java",
			"google-cloud-translate/src/test/java/com/google/cloud/translate/TranslateTest.java",
			"google-cloud-translate/src/test/java/com/google/cloud/translate/TranslationTest.java",
			"google-cloud-translate/src/test/java/com/google/cloud/translate/it/ITTranslateTest.java",
		},
		"aiplatform": {
			"google-cloud-aiplatform/src/test/java/com/google/cloud/location/MockLocations.java",
			"google-cloud-aiplatform/src/test/java/com/google/cloud/location/MockLocationsImpl.java",
			"google-cloud-aiplatform/src/test/java/com/google/iam/v1/MockIAMPolicy.java",
			"google-cloud-aiplatform/src/test/java/com/google/iam/v1/MockIAMPolicyImpl.java",
		},
	}

	javaArtifactIDOverrides = map[string]javaArtifactOverrides{
		"google/datastore/admin/v1": {
			protoArtifactID: "proto-google-cloud-datastore-admin-v1",
			grpcArtifactID:  "grpc-google-cloud-datastore-admin-v1",
		},
		"google/apps/script/type": {
			protoArtifactID: "proto-google-apps-script-type-protos",
		},
		"google/apps/script/type/docs": {
			protoArtifactID: "proto-google-apps-script-type-protos",
		},
		"google/apps/script/type/drive": {
			protoArtifactID: "proto-google-apps-script-type-protos",
		},
		"google/apps/script/type/gmail": {
			protoArtifactID: "proto-google-apps-script-type-protos",
		},
		"google/apps/script/type/sheets": {
			protoArtifactID: "proto-google-apps-script-type-protos",
		},
		"google/apps/script/type/slides": {
			protoArtifactID: "proto-google-apps-script-type-protos",
		},
		"google/spanner/admin/database/v1": {
			protoArtifactID: "proto-google-cloud-spanner-admin-database-v1",
			grpcArtifactID:  "grpc-google-cloud-spanner-admin-database-v1",
		},
		"google/spanner/admin/instance/v1": {
			protoArtifactID: "proto-google-cloud-spanner-admin-instance-v1",
			grpcArtifactID:  "grpc-google-cloud-spanner-admin-instance-v1",
		},
		"google/spanner/executor/v1": {
			protoArtifactID: "proto-google-cloud-spanner-executor-v1",
			grpcArtifactID:  "grpc-google-cloud-spanner-executor-v1",
		},
		"google/devtools/clouderrorreporting/v1beta1": {
			protoArtifactID: "proto-google-cloud-error-reporting-v1beta1",
			grpcArtifactID:  "grpc-google-cloud-error-reporting-v1beta1",
		},
		"google/storage/v2": {
			protoArtifactID: "proto-google-cloud-storage-v2",
			grpcArtifactID:  "grpc-google-cloud-storage-v2",
			gapicArtifactID: "gapic-google-cloud-storage-v2",
		},
		"google/storage/control/v2": {
			protoArtifactID: "proto-google-cloud-storage-control-v2",
			grpcArtifactID:  "grpc-google-cloud-storage-control-v2",
			gapicArtifactID: "google-cloud-storage-control",
		},
	}
)

type javaArtifactOverrides struct {
	protoArtifactID string
	grpcArtifactID  string
	gapicArtifactID string
}
