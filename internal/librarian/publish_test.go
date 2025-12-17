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
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	sidekickconfig "github.com/googleapis/librarian/internal/sidekick/config"
	"github.com/googleapis/librarian/internal/testhelpers"
)

func TestPublish(t *testing.T) {
	ctx := context.Background()

	origRustPublishCrates := rustPublishCrates
	origRustCargoPreFlight := rustCargoPreFlight
	defer func() {
		rustPublishCrates = origRustPublishCrates
		rustCargoPreFlight = origRustCargoPreFlight
	}()

	tests := []struct {
		name                string
		language            string
		dryRun              bool
		skipSemverChecks    bool
		preflightErr        error
		publishErr          error
		wantPreflightCalled bool
		wantPublishCalled   bool
		wantErr             bool
	}{
		{
			name:                "rust success",
			language:            "rust",
			dryRun:              true,
			skipSemverChecks:    true,
			wantPreflightCalled: true,
			wantPublishCalled:   true,
		},
		{
			name:     "unsupported language",
			language: "java",
			wantErr:  true, // error from preflight
		},
		{
			name:                "rust preflight error",
			language:            "rust",
			preflightErr:        errors.New("cargo not found"),
			wantPreflightCalled: true,
			wantPublishCalled:   false,
			wantErr:             true,
		},
		{
			name:                "rust publish error",
			language:            "rust",
			publishErr:          errors.New("publish failed"),
			wantPreflightCalled: true,
			wantPublishCalled:   true,
			wantErr:             true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)
			testhelpers.SetupForVersionBump(t, "v1.0.0")

			var (
				preflightCalled bool
				publishCalled   bool
			)

			rustCargoPreFlight = func(ctx context.Context, config *sidekickconfig.Release) error {
				preflightCalled = true
				return test.preflightErr
			}
			rustPublishCrates = func(ctx context.Context, config *sidekickconfig.Release, dryRun bool, skipSemverChecks bool, lastTag string, files []string) error {
				publishCalled = true
				if lastTag != "v1.0.0" {
					t.Errorf("rustPublishCrates called with wrong lastTag: got %q, want %q", lastTag, "v1.0.0")
				}
				if diff := cmp.Diff([]string{}, files); diff != "" {
					t.Errorf("rustPublishCrates called with wrong files (-want +got):\n%s", diff)
				}
				return test.publishErr
			}

			cfg := &config.Config{
				Language: test.language,
				Release: &config.Release{
					Remote: "origin",
					Branch: "main",
				},
			}

			err := publish(ctx, cfg, test.dryRun, test.skipSemverChecks)

			if (err != nil) != test.wantErr {
				t.Fatalf("publish() error = %v, wantErr %v", err, test.wantErr)
			}

			if preflightCalled != test.wantPreflightCalled {
				t.Errorf("preflightCalled = %v, want %v", preflightCalled, test.wantPreflightCalled)
			}
			if publishCalled != test.wantPublishCalled {
				t.Errorf("publishCalled = %v, want %v", publishCalled, test.wantPublishCalled)
			}
		})
	}
}
