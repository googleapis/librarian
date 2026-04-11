//go:build !google3

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
	"cloud.google.com/go/iam/apiv1/iampb"
	"google.golang.org/genproto/googleapis/api/annotations"
)

// This file provides type aliases and variable mappings for the Open Source (OSS)
// environment. It bridges the differences in import paths and structures between
// OSS and google3 for common Google API annotations and types used in the parser.
//
// Only types and variables with different import paths or structures between OSS
// and google3 should be included here. Do not add aliases for types that are
// identical in both environments.

type resourceDescriptor = annotations.ResourceDescriptor
type routingRule = annotations.RoutingRule
type routingParameter = annotations.RoutingParameter
type resourceReference = annotations.ResourceReference
type httpRule = annotations.HttpRule
type httpRuleGet = annotations.HttpRule_Get
type httpRulePost = annotations.HttpRule_Post
type httpRulePut = annotations.HttpRule_Put
type httpRuleDelete = annotations.HttpRule_Delete
type httpRulePatch = annotations.HttpRule_Patch
type fieldInfo = annotations.FieldInfo
type fieldBehavior = annotations.FieldBehavior

var (
	eResource           = annotations.E_Resource
	eResourceDefinition = annotations.E_ResourceDefinition
	eResourceReference  = annotations.E_ResourceReference
	eRouting            = annotations.E_Routing
	eHttp               = annotations.E_Http
	eDefaultHost        = annotations.E_DefaultHost
	eApiVersion         = annotations.E_ApiVersion
	eFieldInfo          = annotations.E_FieldInfo
	eFieldBehavior      = annotations.E_FieldBehavior

	fileGoogleIamV1IamPolicyProto = iampb.File_google_iam_v1_iam_policy_proto
	fileGoogleIamV1PolicyProto    = iampb.File_google_iam_v1_policy_proto
	fileGoogleIamV1OptionsProto   = iampb.File_google_iam_v1_options_proto
)

const (
	fieldInfoUUID4        = annotations.FieldInfo_UUID4
	fieldBehaviorRequired = annotations.FieldBehavior_REQUIRED
)
