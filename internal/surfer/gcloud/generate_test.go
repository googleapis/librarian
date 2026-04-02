// Copyright 2025 Google LLC
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

package gcloud

import (
	"testing"
)

func TestGenerate_InvalidConfig(t *testing.T) {
	err := Generate(t.Context(), GenerateConfig{
		GcloudConfig: "nonexistent_config.yaml",
	})
	if err == nil {
		t.Error("Generate() error = nil, want error for nonexistent config")
	}
}

func TestGenerate_InvalidModel(t *testing.T) {
	// GcloudConfig is empty, so it might pass (or not, depending on implementation),
	// but Googleapis being nonexistent should definitely fail during model creation.
	err := Generate(t.Context(), GenerateConfig{
		Googleapis: "nonexistent_googleapis_dir",
	})
	if err == nil {
		t.Error("Generate() error = nil, want error for nonexistent googleapis dir")
	}
}
