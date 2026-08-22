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
	"os"

	"github.com/googleapis/librarian/internal/gapic/java/model"
	"go.yaml.in/yaml/v3"
)

type rawGapicYaml struct {
	LanguageSettings map[string]struct {
		PackageName    string            `yaml:"package_name"`
		InterfaceNames map[string]string `yaml:"interface_names"`
	} `yaml:"language_settings"`
	Interfaces []struct {
		Name    string `yaml:"name"`
		Methods []struct {
			Name     string `yaml:"name"`
			Batching *struct {
				Thresholds struct {
					ElementCount int   `yaml:"element_count"`
					RequestByte  int64 `yaml:"request_byte"`
					DelayMillis  int64 `yaml:"delay_millis"`
				} `yaml:"thresholds"`
			} `yaml:"batching"`
			LongRunning *struct {
				InitialPollDelayMillis int64   `yaml:"initial_poll_delay_millis"`
				PollDelayMultiplier    float64 `yaml:"poll_delay_multiplier"`
				MaxPollDelayMillis     int64   `yaml:"max_poll_delay_millis"`
				TotalPollTimeoutMillis int64   `yaml:"total_poll_timeout_millis"`
			} `yaml:"long_running"`
		} `yaml:"methods"`
	} `yaml:"interfaces"`
}

// ParseGapicYaml parses gapic.yaml for batching, LRO retry settings, and language settings.
func ParseGapicYaml(filePath string) ([]*model.GapicBatchingSettings, []*model.GapicLroRetrySettings, *model.GapicLanguageSettings, error) {
	if filePath == "" {
		return nil, nil, nil, nil
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, nil, err
	}
	return ParseGapicYamlBytes(content)
}

// ParseGapicYamlBytes parses YAML content for gapic.yaml.
func ParseGapicYamlBytes(content []byte) ([]*model.GapicBatchingSettings, []*model.GapicLroRetrySettings, *model.GapicLanguageSettings, error) {
	var raw rawGapicYaml
	if err := yaml.Unmarshal(content, &raw); err != nil {
		return nil, nil, nil, err
	}

	var batching []*model.GapicBatchingSettings
	var lroSettings []*model.GapicLroRetrySettings
	var langSettings *model.GapicLanguageSettings

	if javaSettings, ok := raw.LanguageSettings["java"]; ok {
		langSettings = &model.GapicLanguageSettings{
			PackageName:    javaSettings.PackageName,
			InterfaceNames: javaSettings.InterfaceNames,
		}
	}

	for _, iface := range raw.Interfaces {
		for _, m := range iface.Methods {
			fullMethod := iface.Name + "." + m.Name
			if m.Batching != nil {
				batching = append(batching, &model.GapicBatchingSettings{
					MethodName:            fullMethod,
					ElementCountThreshold: m.Batching.Thresholds.ElementCount,
					RequestByteThreshold:  m.Batching.Thresholds.RequestByte,
					DelayThresholdMillis:  m.Batching.Thresholds.DelayMillis,
				})
			}
			if m.LongRunning != nil {
				lroSettings = append(lroSettings, &model.GapicLroRetrySettings{
					MethodName:             fullMethod,
					InitialPollDelayMillis: m.LongRunning.InitialPollDelayMillis,
					PollDelayMultiplier:    m.LongRunning.PollDelayMultiplier,
					MaxPollDelayMillis:     m.LongRunning.MaxPollDelayMillis,
					TotalPollTimeoutMillis: m.LongRunning.TotalPollTimeoutMillis,
				})
			}
		}
	}

	return batching, lroSettings, langSettings, nil
}
