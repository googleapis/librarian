// Copyright 2024 Google LLC
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

package sidekick

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"
)

const (
	// projectRoot is the root of the google-cloud-rust. The golden files for
	// these tests depend on code in ../../auth and ../../src/gax.
	projectRoot = "../.."
)

var (
	testdataDir, _             = filepath.Abs("../testdata")
	googleapisRoot             = fmt.Sprintf("%s/googleapis", testdataDir)
	outputDir                  = fmt.Sprintf("%s/test-only", testdataDir)
	secretManagerServiceConfig = "googleapis/google/cloud/secretmanager/v1/secretmanager_v1.yaml"
	specificationSource        = fmt.Sprintf("%s/openapi/secretmanager_openapi_v1.json", testdataDir)
)

func requireCommand(t *testing.T, command string) {
	t.Helper()
	if _, err := exec.LookPath(command); err != nil {
		t.Skipf("skipping test because %s is not installed", command)
	}
}

func requireProtoc(t *testing.T) {
	requireCommand(t, "protoc")
}

func requireCargo(t *testing.T) {
	requireCommand(t, "cargo")
}
