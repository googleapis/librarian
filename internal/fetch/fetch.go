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

package fetch

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
)

func GetSha256(query string) (string, error) {
	response, err := http.Get(query)
	if err != nil {
		return "", err
	}
	if response.StatusCode >= 300 {
		return "", fmt.Errorf("http error in download %s", response.Status)
	}
	defer response.Body.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, response.Body); err != nil {
		return "", err
	}
	got := fmt.Sprintf("%x", hasher.Sum(nil))
	return got, nil
}

func GetLatestSha(query string) (string, error) {
	client := &http.Client{}
	request, err := http.NewRequest(http.MethodGet, query, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "application/vnd.github.VERSION.sha")
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	if response.StatusCode >= 300 {
		return "", fmt.Errorf("http error in download %s", response.Status)
	}
	defer response.Body.Close()
	contents, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	return string(contents), nil
}
