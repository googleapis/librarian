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

type overrideKey struct {
	libraryName string
	apiPath     string
}

var (
	excludedSamplesLibraries = map[string]bool{
		"bigquerystorage":   true,
		"datastore":         true,
		"logging":           true,
		"storage":           true,
		"spanner":           true,
		"containeranalysis": true,
		"common-protos":     true,
		"grafeas":           true,
		"iam":               true,
		"iam-policy":        true,
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
		"showcase": {
			"gapic-showcase/src/test",
			"gapic-showcase/src/main/java/com/google/showcase/v1beta1/Version.java",
		},
	}

	javaArtifactIDOverrides = map[overrideKey]javaArtifactOverrides{
		{apiPath: "google/datastore/admin/v1"}: {
			protoArtifactID: "proto-google-cloud-datastore-admin-v1",
			grpcArtifactID:  "grpc-google-cloud-datastore-admin-v1",
		},
		{apiPath: "google/api"}: {
			protoArtifactID: "proto-google-common-protos",
		},
		{apiPath: "google/apps/card/v1"}: {
			protoArtifactID: "proto-google-common-protos",
		},
		{apiPath: "google/apps/script/type"}: {
			protoArtifactID: "proto-google-apps-script-type-protos",
		},
		{apiPath: "google/apps/script/type/docs"}: {
			protoArtifactID: "proto-google-apps-script-type-protos",
		},
		{apiPath: "google/apps/script/type/drive"}: {
			protoArtifactID: "proto-google-apps-script-type-protos",
		},
		{apiPath: "google/apps/script/type/gmail"}: {
			protoArtifactID: "proto-google-apps-script-type-protos",
		},
		{apiPath: "google/apps/script/type/sheets"}: {
			protoArtifactID: "proto-google-apps-script-type-protos",
		},
		{apiPath: "google/apps/script/type/slides"}: {
			protoArtifactID: "proto-google-apps-script-type-protos",
		},
		{libraryName: "iam", apiPath: "google/iam/v1"}: {
			protoArtifactID: "proto-google-iam-v1",
			grpcArtifactID:  "grpc-google-iam-v1",
		},
		{libraryName: "iam", apiPath: "google/iam/v2"}: {
			protoArtifactID: "proto-google-iam-v2",
			grpcArtifactID:  "grpc-google-iam-v2",
		},
		{libraryName: "iam", apiPath: "google/iam/v3"}: {
			protoArtifactID: "proto-google-iam-v3",
			grpcArtifactID:  "grpc-google-iam-v3",
		},
		{libraryName: "iam", apiPath: "google/iam/v2beta"}: {
			protoArtifactID: "proto-google-iam-v2beta",
			grpcArtifactID:  "grpc-google-iam-v2beta",
		},
		{libraryName: "iam", apiPath: "google/iam/v3beta"}: {
			protoArtifactID: "proto-google-iam-v3beta",
			grpcArtifactID:  "grpc-google-iam-v3beta",
		},
		{apiPath: "google/cloud"}: {
			protoArtifactID: "proto-google-common-protos",
		},
		{apiPath: "google/cloud/audit"}: {
			protoArtifactID: "proto-google-common-protos",
		},
		{apiPath: "google/cloud/location"}: {
			protoArtifactID: "proto-google-common-protos",
			grpcArtifactID:  "grpc-google-common-protos",
		},
		{apiPath: "google/geo/type"}: {
			protoArtifactID: "proto-google-common-protos",
		},
		{apiPath: "google/logging/type"}: {
			protoArtifactID: "proto-google-common-protos",
		},
		{apiPath: "google/longrunning"}: {
			protoArtifactID: "proto-google-common-protos",
			grpcArtifactID:  "grpc-google-common-protos",
		},
		{apiPath: "google/rpc"}: {
			protoArtifactID: "proto-google-common-protos",
		},
		{apiPath: "google/rpc/context"}: {
			protoArtifactID: "proto-google-common-protos",
		},
		{apiPath: "google/shopping/type"}: {
			protoArtifactID: "proto-google-common-protos",
		},
		{apiPath: "google/type"}: {
			protoArtifactID: "proto-google-common-protos",
		},
		{apiPath: "google/spanner/admin/database/v1"}: {
			protoArtifactID: "proto-google-cloud-spanner-admin-database-v1",
			grpcArtifactID:  "grpc-google-cloud-spanner-admin-database-v1",
		},
		{apiPath: "google/spanner/admin/instance/v1"}: {
			protoArtifactID: "proto-google-cloud-spanner-admin-instance-v1",
			grpcArtifactID:  "grpc-google-cloud-spanner-admin-instance-v1",
		},
		{apiPath: "google/spanner/executor/v1"}: {
			protoArtifactID: "proto-google-cloud-spanner-executor-v1",
			grpcArtifactID:  "grpc-google-cloud-spanner-executor-v1",
		},
		{apiPath: "google/devtools/clouderrorreporting/v1beta1"}: {
			protoArtifactID: "proto-google-cloud-error-reporting-v1beta1",
			grpcArtifactID:  "grpc-google-cloud-error-reporting-v1beta1",
		},
		{apiPath: "google/storage/v2"}: {
			protoArtifactID: "proto-google-cloud-storage-v2",
			grpcArtifactID:  "grpc-google-cloud-storage-v2",
			gapicArtifactID: "gapic-google-cloud-storage-v2",
		},
		{apiPath: "google/storage/control/v2"}: {
			protoArtifactID: "proto-google-cloud-storage-control-v2",
			grpcArtifactID:  "grpc-google-cloud-storage-control-v2",
			gapicArtifactID: "google-cloud-storage-control",
		},
		{apiPath: "schema/google/showcase/v1beta1"}: {
			protoArtifactID: "proto-gapic-showcase-v1beta1",
			grpcArtifactID:  "grpc-gapic-showcase-v1beta1",
		},
	}

	javaTransportOverrides = map[string]string{
		//This is added here instead of sdk.yaml change because this is
		//a proto-only library and transport does not affect Java code generated.
		"alloydb-connectors": "grpc",
		"common-protos":      "grpc",
	}

	apiShortnameOverrides = map[string]string{
		"common-protos": "common-protos",
	}

	skipAPIID = map[string]bool{
		"google-auth-library": true,
		"showcase":            true,
		"iam":                 true,
		"api-common":          true,
		"common-protos":       true,
		"gax":                 true,
		"core":                true,
	}

	skipPOMUpdates = map[string]bool{
		"grafeas":       true,
		"common-protos": true,
	}

	monolithicJavaAPIs = map[string]bool{
		"grafeas/v1": true,
	}

	javaAdditionalProtosOverrides = map[string][]string{
		"schema/google/showcase/v1beta1": {
			"google/cloud/location/locations.proto",
			"google/iam/v1/iam_policy.proto",
		},
		"google/cloud/filestore": {
			"google/cloud/common/operation_metadata.proto",
		},
		"google/cloud/oslogin": {
			"google/cloud/oslogin/common/common.proto",
		},
	}
)

type javaArtifactOverrides struct {
	protoArtifactID string
	grpcArtifactID  string
	gapicArtifactID string
}
