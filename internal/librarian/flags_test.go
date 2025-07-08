// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package librarian

import (
	"flag"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"strings"
	"testing"
)

func TestAddFlagHostMount(t *testing.T) {
	for _, test := range []struct {
		name         string
		initialValue string
		flagValue    string
		want         string
		wantErr      bool
	}{
		{
			name:         "default value, no flag provided",
			initialValue: "", // Simulates initial empty config value
			flagValue:    "", // No flag provided means it remains default
			want:         "",
			wantErr:      false,
		},
		{
			name:         "flag provided with valid value",
			initialValue: "",
			flagValue:    "hostDir:/localDir",
			want:         "hostDir:/localDir",
			wantErr:      false,
		},
		{
			name:         "flag provided with invalid format (missing colon)",
			initialValue: "",
			flagValue:    "hostDir/localDir",
			wantErr:      true, // Expecting a panic
		},
		{
			name:         "flag provided with invalid format (empty host dir)",
			initialValue: "",
			flagValue:    ":/localDir",
			wantErr:      true, // Expecting a panic due to `strings.Split` result
		},
		{
			name:         "flag provided with invalid format (empty local dir)",
			initialValue: "",
			flagValue:    "hostDir:",
			wantErr:      true, // Expecting a panic due to `strings.Split` result
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{HostMount: test.initialValue}
			fs := flag.NewFlagSet(test.name, flag.ContinueOnError)
			args := []string{}
			if test.flagValue != "" {
				args = append(args, "-host-mount="+test.flagValue)
			}
			if err := fs.Parse(args); err != nil {
				t.Fatalf("fs.Parse failed: %v", err)
			}

			if test.wantErr {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("The code did not panic for invalid host-mount format: %s", test.flagValue)
					} else {
						// Check the panic message
						panicMsg := fmt.Sprintf("%v", r)
						if !strings.Contains(panicMsg, "The value of flag host-mount should be {host-dir}:{local-dir}") {
							t.Errorf("Unexpected panic message: %s", panicMsg)
						}
					}
				}()
				addFlagHostMount(fs, cfg)
				// If we reach here, it means no panic occurred, which is an error for wantErr=true
				t.Fatalf("Expected panic for input %q, but no panic occurred.", test.flagValue)
			} else {
				addFlagHostMount(fs, cfg)
				if diff := cmp.Diff(test.want, cfg.HostMount); diff != "" {
					t.Errorf("HostMount mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
