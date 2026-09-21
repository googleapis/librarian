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

package python

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/testhelper"
)

const (
	minCredentialsEmittedFiles = 17
	minRedisEmittedFiles       = 17
	minAssetEmittedFiles       = 18
)

func TestGoldenParity(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")

	testdataDir, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	protosDir := filepath.Join(testdataDir, "protos")
	goldenBaseDir := filepath.Join(testdataDir, "golden")

	for _, test := range []struct {
		name            string
		specSource      string
		serviceConfig   string
		libraryName     string
		packageVersion  string
		defaultVersion  string
		goldenRelDir    string
		expectedService string
		minEmittedFiles int
		requiredFiles   []string
	}{
		{
			name:            "credentials",
			specSource:      "google/iam/credentials/v1",
			serviceConfig:   "google/iam/credentials/v1/iamcredentials_v1.yaml",
			libraryName:     "google-iam-credentials",
			packageVersion:  "0.0.0",
			defaultVersion:  "v1",
			goldenRelDir:    "credentials",
			expectedService: ".google.iam.credentials.v1.IAMCredentials",
			minEmittedFiles: minCredentialsEmittedFiles,
			requiredFiles: []string{
				"google/iam/credentials_v1/_compat.py",
				"google/iam/credentials_v1/services/iam_credentials/client.py",
				"google/iam/credentials_v1/services/iam_credentials/async_client.py",
				"google/iam/credentials_v1/services/iam_credentials/transports/base.py",
				"google/iam/credentials_v1/services/iam_credentials/transports/grpc.py",
				"google/iam/credentials_v1/services/iam_credentials/transports/grpc_asyncio.py",
				"google/iam/credentials_v1/services/iam_credentials/transports/rest_base.py",
				"google/iam/credentials_v1/services/iam_credentials/transports/rest.py",
				"google/iam/credentials_v1/services/iam_credentials/transports/__init__.py",
				"google/iam/credentials_v1/types/common.py",
				"google/iam/credentials_v1/types/iamcredentials.py",
			},
		},
		{
			name:            "redis",
			specSource:      "google/cloud/redis/v1",
			serviceConfig:   "google/cloud/redis/v1/redis_v1.yaml",
			libraryName:     "google-cloud-redis",
			packageVersion:  "0.0.0",
			defaultVersion:  "v1",
			goldenRelDir:    "redis",
			expectedService: ".google.cloud.redis.v1.CloudRedis",
			minEmittedFiles: minRedisEmittedFiles,
			requiredFiles: []string{
				"google/cloud/redis_v1/_compat.py",
				"google/cloud/redis_v1/services/cloud_redis/client.py",
				"google/cloud/redis_v1/services/cloud_redis/async_client.py",
				"google/cloud/redis_v1/services/cloud_redis/pagers.py",
				"google/cloud/redis_v1/services/cloud_redis/transports/base.py",
				"google/cloud/redis_v1/services/cloud_redis/transports/grpc.py",
				"google/cloud/redis_v1/services/cloud_redis/transports/grpc_asyncio.py",
				"google/cloud/redis_v1/services/cloud_redis/transports/rest_base.py",
				"google/cloud/redis_v1/services/cloud_redis/transports/rest.py",
				"google/cloud/redis_v1/services/cloud_redis/transports/__init__.py",
				"google/cloud/redis_v1/types/cloud_redis.py",
			},
		},
		{
			name:            "asset",
			specSource:      "google/cloud/asset/v1",
			serviceConfig:   "google/cloud/asset/v1/cloudasset_v1.yaml",
			libraryName:     "google-cloud-asset",
			packageVersion:  "1.2.99",
			defaultVersion:  "v1",
			goldenRelDir:    "asset",
			expectedService: ".google.cloud.asset.v1.AssetService",
			minEmittedFiles: minAssetEmittedFiles,
			requiredFiles: []string{
				"google/cloud/asset_v1/_compat.py",
				"google/cloud/asset_v1/services/asset_service/client.py",
				"google/cloud/asset_v1/services/asset_service/async_client.py",
				"google/cloud/asset_v1/services/asset_service/pagers.py",
				"google/cloud/asset_v1/services/asset_service/transports/base.py",
				"google/cloud/asset_v1/services/asset_service/transports/grpc.py",
				"google/cloud/asset_v1/services/asset_service/transports/grpc_asyncio.py",
				"google/cloud/asset_v1/services/asset_service/transports/rest_base.py",
				"google/cloud/asset_v1/services/asset_service/transports/rest.py",
				"google/cloud/asset_v1/services/asset_service/transports/__init__.py",
				"google/cloud/asset_v1/types/asset_service.py",
				"google/cloud/asset_v1/types/assets.py",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()

			cfg := &parser.ModelConfig{
				Language:            config.LanguagePython,
				SpecificationFormat: config.SpecProtobuf,
				ServiceConfig:       test.serviceConfig,
				SpecificationSource: test.specSource,
				Source: &sources.SourceConfig{
					Sources: &sources.Sources{
						Googleapis: protosDir,
					},
					ActiveRoots: []string{"googleapis"},
				},
			}

			model, err := parser.CreateModel(cfg)
			if err != nil {
				t.Fatal(err)
			}
			if model == nil {
				t.Fatal("parser.CreateModel returned nil model")
			}
			if model.Service(test.expectedService) == nil {
				t.Errorf("model missing expected service %q", test.expectedService)
			}

			lib := &config.Library{
				Name:          test.libraryName,
				Output:        outDir,
				Version:       test.packageVersion,
				CopyrightYear: "2026",
				Roots:         []string{protosDir},
				APIs: []*config.API{
					{Path: test.specSource},
				},
				Python: &config.PythonPackage{
					DefaultVersion: test.defaultVersion,
					PythonDefault: config.PythonDefault{
						Generator: "sidekick",
					},
				},
			}

			if err := Generate(t.Context(), model, outDir, lib); err != nil {
				t.Fatal(err)
			}

			var emittedFiles []string
			err = filepath.WalkDir(outDir, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}

				relPath, err := filepath.Rel(outDir, path)
				if err != nil {
					return err
				}
				if strings.HasPrefix(relPath, "..") || filepath.IsAbs(relPath) {
					t.Errorf("emitted file %s escapes outDir %s", relPath, outDir)
					return nil
				}

				relSlash := filepath.ToSlash(relPath)
				if !strings.HasPrefix(relSlash, "google/") {
					return nil
				}
				emittedFiles = append(emittedFiles, relSlash)

				gotBytes, err := os.ReadFile(path)
				if err != nil {
					t.Error(err)
					return nil
				}
				goldenPath := filepath.Join(goldenBaseDir, test.goldenRelDir, relPath)
				if _, err := os.Stat(goldenPath); errors.Is(err, fs.ErrNotExist) {
					t.Errorf("emitted file %s does not exist in golden directory %s", relPath, filepath.Join(goldenBaseDir, test.goldenRelDir))
					return nil
				}
				wantBytes, err := os.ReadFile(goldenPath)
				if err != nil {
					t.Error(err)
					return nil
				}
				if diff := cmp.Diff(string(wantBytes), string(gotBytes)); diff != "" {
					t.Logf("[%s] file: %s", test.name, relPath)
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}

				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(emittedFiles) < test.minEmittedFiles {
				t.Errorf("emitted files count for %s = %d, want at least %d", test.name, len(emittedFiles), test.minEmittedFiles)
			}
			for _, req := range test.requiredFiles {
				if !slices.Contains(emittedFiles, req) {
					t.Errorf("missing required golden file %s for %s", req, test.name)
				}
			}
		})
	}
}
