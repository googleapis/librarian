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

import (
	"github.com/googleapis/librarian/internal/config"
)

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
		"bigtable":          true,
		"firestore":         true,
		"pubsub":            true,
	}

	keepOverride = map[string][]string{
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
		"spanner": {
			"google-cloud-spanner-executor/src/main/java/com/google/cloud/spanner/executor/v1/stub/Version.java",
			"google-cloud-spanner/src/main/java/com/google/cloud/spanner/admin/database/v1/stub/Version.java",
			"google-cloud-spanner/src/main/java/com/google/cloud/spanner/admin/instance/v1/stub/Version.java",
			"google-cloud-spanner/src/main/java/com/google/cloud/spanner/v1/stub/Version.java",
			"google-cloud-spanner/src/main/resources/META-INF/native-image/com.google.cloud.spanner/reflect-config.json",
			"proto-google-cloud-spanner-admin-database-v1/src/main/java/com/google/spanner/admin/database/v1/CryptoKeyName.java",
			"proto-google-cloud-spanner-admin-database-v1/src/main/java/com/google/spanner/admin/database/v1/CryptoKeyVersionName.java",
		},
	}

	keepAppends = map[string][]string{
		"firestore": {
			"google-cloud-firestore/src/main/resources/META-INF/native-image/com.google.cloud/google-cloud-firestore/reflect-config.json",
			"google-cloud-firestore/src/test/resources/META-INF/native-image/reflect-config.json",
		},
	}

	javaArtifactIDOverrides = map[overrideKey]javaArtifactOverrides{
		{apiPath: "google/datastore/admin/v1"}: {
			protoArtifactID: "proto-google-cloud-datastore-admin-v1",
			grpcArtifactID:  "grpc-google-cloud-datastore-admin-v1",
		},
		{apiPath: "google/firestore/admin/v1"}: {
			protoArtifactID: "proto-google-cloud-firestore-admin-v1",
			grpcArtifactID:  "grpc-google-cloud-firestore-admin-v1",
			gapicArtifactID: "google-cloud-firestore-admin",
		},
		{apiPath: "google/firestore/bundle"}: {
			protoArtifactID: "proto-google-cloud-firestore-bundle-v1",
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
			gapicArtifactID: "google-cloud-spanner-executor",
		},
		{apiPath: "google/devtools/clouderrorreporting/v1beta1"}: {
			protoArtifactID: "proto-google-cloud-error-reporting-v1beta1",
			grpcArtifactID:  "grpc-google-cloud-error-reporting-v1beta1",
		},
		{apiPath: "google/bigtable/admin/v2"}: {
			protoArtifactID: "proto-google-cloud-bigtable-admin-v2",
			grpcArtifactID:  "grpc-google-cloud-bigtable-admin-v2",
			gapicArtifactID: "google-cloud-bigtable",
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
		"firestore":     true,
	}

	monolithicJavaAPIs = map[string]bool{
		"grafeas/v1": true,
	}

	javaAdditionalProtosOverrides = map[string][]*config.AdditionalProto{
		"schema/google/showcase/v1beta1": {
			{Path: "google/cloud/location/locations.proto"},
			{Path: "google/iam/v1/iam_policy.proto"},
		},
		"google/cloud/filestore": {
			{
				Path:                 "google/cloud/common/operation_metadata.proto",
				GenerateProtoClasses: true,
			},
		},
		"google/cloud/oslogin": {
			{
				Path:                 "google/cloud/oslogin/common/common.proto",
				GenerateProtoClasses: true,
				CopyToOutput:         true,
			},
		},
	}

	javaAPIOverrides = map[overrideKey]struct {
		GenerateGAPIC         *bool
		GenerateProto         *bool
		GenerateGRPC          *bool
		GenerateResourceNames *bool
	}{
		// IAM v2 and v3 special overrides
		{libraryName: "iam", apiPath: "google/iam/v2"}: {
			GenerateGAPIC:         new(false),
			GenerateProto:         new(true),
			GenerateGRPC:          new(true),
			GenerateResourceNames: new(true),
		},
		{libraryName: "iam", apiPath: "google/iam/v2beta"}: {
			GenerateGAPIC:         new(false),
			GenerateProto:         new(true),
			GenerateGRPC:          new(true),
			GenerateResourceNames: new(true),
		},
		{libraryName: "iam", apiPath: "google/iam/v3"}: {
			GenerateGAPIC:         new(false),
			GenerateProto:         new(true),
			GenerateGRPC:          new(true),
			GenerateResourceNames: new(true),
		},
		{libraryName: "iam", apiPath: "google/iam/v3beta"}: {
			GenerateGAPIC:         new(false),
			GenerateProto:         new(true),
			GenerateGRPC:          new(true),
			GenerateResourceNames: new(true),
		},
		{libraryName: "iam-policy", apiPath: "google/iam/v2"}: {
			GenerateGAPIC:         new(true),
			GenerateProto:         new(false),
			GenerateGRPC:          new(false),
			GenerateResourceNames: new(false),
		},
		{libraryName: "iam-policy", apiPath: "google/iam/v2beta"}: {
			GenerateGAPIC:         new(true),
			GenerateProto:         new(false),
			GenerateGRPC:          new(false),
			GenerateResourceNames: new(false),
		},
		{libraryName: "iam-policy", apiPath: "google/iam/v3"}: {
			GenerateGAPIC:         new(true),
			GenerateProto:         new(false),
			GenerateGRPC:          new(false),
			GenerateResourceNames: new(false),
		},
		{libraryName: "iam-policy", apiPath: "google/iam/v3beta"}: {
			GenerateGAPIC:         new(true),
			GenerateProto:         new(false),
			GenerateGRPC:          new(false),
			GenerateResourceNames: new(false),
		},
		// orgpolicy v1
		{libraryName: "orgpolicy", apiPath: "google/cloud/orgpolicy/v1"}: {
			GenerateGAPIC:         new(false),
			GenerateGRPC:          new(false),
			GenerateResourceNames: new(false),
		},
		// gsuite-addons types
		{libraryName: "gsuite-addons", apiPath: "google/apps/script/type"}: {
			GenerateGAPIC:         new(false),
			GenerateGRPC:          new(false),
			GenerateResourceNames: new(false),
		},
		{libraryName: "gsuite-addons", apiPath: "google/apps/script/type/docs"}: {
			GenerateGAPIC:         new(false),
			GenerateGRPC:          new(false),
			GenerateResourceNames: new(false),
		},
		{libraryName: "gsuite-addons", apiPath: "google/apps/script/type/drive"}: {
			GenerateGAPIC:         new(false),
			GenerateGRPC:          new(false),
			GenerateResourceNames: new(false),
		},
		{libraryName: "gsuite-addons", apiPath: "google/apps/script/type/gmail"}: {
			GenerateGAPIC:         new(false),
			GenerateGRPC:          new(false),
			GenerateResourceNames: new(false),
		},
		{libraryName: "gsuite-addons", apiPath: "google/apps/script/type/sheets"}: {
			GenerateGAPIC:         new(false),
			GenerateGRPC:          new(false),
			GenerateResourceNames: new(false),
		},
		{libraryName: "gsuite-addons", apiPath: "google/apps/script/type/slides"}: {
			GenerateGAPIC:         new(false),
			GenerateGRPC:          new(false),
			GenerateResourceNames: new(false),
		},
	}
)

type javaArtifactOverrides struct {
	protoArtifactID string
	grpcArtifactID  string
	gapicArtifactID string
}
