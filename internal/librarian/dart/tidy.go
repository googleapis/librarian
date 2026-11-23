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

package dart

import (
	"errors"

	"github.com/googleapis/librarian/internal/config"
)

// ErrSDKVersionMissing is returned when default.dart.sdk_version is not set.
var ErrSDKVersionMissing = errors.New("dart sdk_version not found in config")

// Validate checks that the Dart-specific configuration is valid.
func Validate(cfg *config.Config) error {
	if cfg.Default == nil || cfg.Default.Dart == nil || cfg.Default.Dart.SDKVersion == "" {
		return ErrSDKVersionMissing
	}
	return nil
}
