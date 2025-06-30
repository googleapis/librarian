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

package errors

import "fmt"

// LibError represents a structured application error with an optional cause.
type LibError struct {
	Reason string
	Cause  error
}

// Error returns the formatted error message for LibError.
// If a cause is present, it is included in the message.
func (e *LibError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Reason, e.Cause)
	}
	return e.Reason
}

// Unwrap returns the underlying cause of the LibError, enabling error unwrapping.
func (e *LibError) Unwrap() error {
	return e.Cause
}

// CustomError creates a new LibError using the provided template and formatting parameters.
// This is used when no underlying cause needs to be tracked.
func CustomError(template string, params ...interface{}) error {
	return &LibError{
		Reason: fmt.Sprintf(template, params...),
	}
}

// CustomErrorWrap creates a new LibError with a formatted message and wraps the provided cause.
// This supports Go's errors.Is and errors.Unwrap behavior.
func CustomErrorWrap(cause error, template string, params ...interface{}) error {
	return &LibError{
		Reason: fmt.Sprintf(template, params...),
		Cause:  cause,
	}
}
