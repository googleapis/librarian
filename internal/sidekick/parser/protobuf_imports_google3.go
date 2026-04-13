//go:build google3

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

package parser

import (
	annotationspb "google3/google/api/annotations_go_proto"
	clientpb "google3/google/api/client_go_proto"
	fbpb "google3/google/api/field_behavior_go_proto"
	fipb "google3/google/api/field_info_go_proto"
	resourcepb "google3/google/api/resource_go_proto"
	routingpb "google3/google/api/routing_go_proto"
	iampb "google3/google/iam/v1/iam_policy_go_proto"
	optionspb "google3/google/iam/v1/options_go_proto"
	policypb "google3/google/iam/v1/policy_go_proto"
)

// This file provides type aliases and variable mappings for the google3 environment.
// It bridges the differences in import paths and structures between OSS and
// google3 for common Google API annotations and types used in the parser.
//
// Only types and variables with different import paths or structures between OSS
// and google3 should be included here. Do not add aliases for types that are
// identical in both environments.

type resourceDescriptor = resourcepb.ResourceDescriptor
type routingRule = routingpb.RoutingRule
type routingParameter = routingpb.RoutingParameter
type resourceReference = resourcepb.ResourceReference
type httpRule = annotationspb.HttpRule
type httpRuleGet = annotationspb.HttpRule_Get
type httpRulePost = annotationspb.HttpRule_Post
type httpRulePut = annotationspb.HttpRule_Put
type httpRuleDelete = annotationspb.HttpRule_Delete
type httpRulePatch = annotationspb.HttpRule_Patch
type fieldInfo = fipb.FieldInfo
type fieldBehavior = fbpb.FieldBehavior

var (
	eResource           = resourcepb.E_Resource
	eResourceDefinition = resourcepb.E_ResourceDefinition
	eResourceReference  = resourcepb.E_ResourceReference
	eRouting            = routingpb.E_Routing
	eHttp               = annotationspb.E_Http
	eDefaultHost        = clientpb.E_DefaultHost
	eApiVersion         = clientpb.E_ApiVersion
	eFieldInfo          = fipb.E_FieldInfo
	eFieldBehavior      = fbpb.E_FieldBehavior

	fileGoogleIamV1IamPolicyProto = iampb.File_google_iam_v1_iam_policy_proto
	fileGoogleIamV1PolicyProto    = policypb.File_google_iam_v1_policy_proto
	fileGoogleIamV1OptionsProto   = optionspb.File_google_iam_v1_options_proto
)

const (
	fieldInfoUUID4        = fipb.FieldInfo_UUID4
	fieldBehaviorRequired = fbpb.FieldBehavior_REQUIRED
)
