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

package protoparser

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"

	"github.com/googleapis/librarian/internal/gapic/java/model"
)

type rawServiceConfig struct {
	MethodConfig []rawMethodConfig `json:"methodConfig"`
	RetryPolicy  map[string]any    `json:"retryPolicy"`
}

type rawMethodConfig struct {
	Name []struct {
		Service string `json:"service"`
		Method  string `json:"method"`
	} `json:"name"`
	Timeout              string          `json:"timeout"`
	RetryPolicy          *rawRetryPolicy `json:"retryPolicy"`
	RetryOrHedgingPolicy string          `json:"retryOrHedgingPolicy"`
}

type rawRetryPolicy struct {
	MaxAttempts          int      `json:"maxAttempts"`
	InitialBackoff       string   `json:"initialBackoff"`
	MaxBackoff           string   `json:"maxBackoff"`
	BackoffMultiplier    float64  `json:"backoffMultiplier"`
	RetryableStatusCodes []string `json:"retryableStatusCodes"`
}

// ParseServiceConfigJSON parses a grpc_service_config.json file into GapicServiceConfig.
func ParseServiceConfigJSON(filePath string) (*model.GapicServiceConfig, error) {
	if filePath == "" {
		return nil, nil
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return ParseServiceConfigJSONBytes(content)
}

// ParseServiceConfigJSONBytes parses JSON bytes into GapicServiceConfig.
func ParseServiceConfigJSONBytes(content []byte) (*model.GapicServiceConfig, error) {
	var raw rawServiceConfig
	if err := json.Unmarshal(content, &raw); err != nil {
		return nil, err
	}

	cfg := &model.GapicServiceConfig{
		RetryCodes:    make(map[string][]string),
		RetryParams:   make(map[string]*model.RetryParam),
		MethodConfigs: make(map[string]*model.MethodConfig),
	}

	for _, mc := range raw.MethodConfig {
		timeoutMillis := parseDurationMillis(mc.Timeout)

		var retryPolicyName string
		if mc.RetryPolicy != nil {
			retryPolicyName = "retry_policy"
			if len(mc.RetryPolicy.RetryableStatusCodes) > 0 {
				cfg.RetryCodes[retryPolicyName] = mc.RetryPolicy.RetryableStatusCodes
			}
			cfg.RetryParams[retryPolicyName] = &model.RetryParam{
				InitialRetryDelayMillis: parseDurationMillis(mc.RetryPolicy.InitialBackoff),
				RetryDelayMultiplier:    mc.RetryPolicy.BackoffMultiplier,
				MaxRetryDelayMillis:     parseDurationMillis(mc.RetryPolicy.MaxBackoff),
				InitialRpcTimeoutMillis: timeoutMillis,
				RpcTimeoutMultiplier:    1.0,
				MaxRpcTimeoutMillis:     timeoutMillis,
				TotalTimeoutMillis:      timeoutMillis,
			}
		}

		for _, name := range mc.Name {
			fullMethod := name.Service
			if name.Method != "" {
				fullMethod = name.Service + "." + name.Method
			}
			cfg.MethodConfigs[fullMethod] = &model.MethodConfig{
				RetryPolicyName: retryPolicyName,
				TimeoutMillis:   timeoutMillis,
			}
		}
	}

	return cfg, nil
}

func parseDurationMillis(dur string) int64 {
	if dur == "" {
		return 0
	}
	dur = strings.TrimSuffix(dur, "s")
	secs, err := strconv.ParseFloat(dur, 64)
	if err != nil {
		return 0
	}
	return int64(secs * 1000)
}
