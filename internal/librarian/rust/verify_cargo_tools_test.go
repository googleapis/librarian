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

package rust

import (
	"testing"

	"github.com/googleapis/librarian/internal/testhelper"
)

func TestCargoPreFlightSuccess(t *testing.T) {
	tools := []Tool{
		{Name: "cargo-semver-checks"},
	}
	if err := CargoPreFlight(t.Context(), "cargo", tools); err != nil {
		t.Fatal(err)
	}
}

func TestCargoPreFlightBadCargo(t *testing.T) {
	tools := []Tool{
		{Name: "cargo-semver-checks"},
	}
	if err := CargoPreFlight(t.Context(), "not-a-valid-cargo", tools); err == nil {
		t.Error("expected an error, got none")
	}
}

func TestCargoPreFlightBadTool(t *testing.T) {
	testhelper.RequireCommand(t, "cargo")
	tools := []Tool{
		{Name: "not-a-valid-tool", Version: "0.0.1"},
	}
	if err := CargoPreFlight(t.Context(), "cargo", tools); err == nil {
		t.Error("expected an error, got none")
	}
}
