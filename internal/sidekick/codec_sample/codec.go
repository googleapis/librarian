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

package codec_sample

import (
	"fmt"
	"time"
)

type codec struct {
	// New generators initialize a field like this one from the library
	// configuration. That makes the generation output stable when no input has
	// changed.
	//
	// Older generators update the copyright year on each generation. That is
	// also valid, just a little more churn.
	CopyrightYear string
}

func newCodec() (*codec, error) {
	return &codec{
		CopyrightYear: fmt.Sprintf("%04d", time.Now().Year()),
	}, nil
}
