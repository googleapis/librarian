#!/bin/bash
# Copyright 2025 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


export CONVERTER_IMAGE_AND_TAG="${1:-discovery-converter:build}"
export LIBRARIAN_STD_IMAGE_AND_TAG="${2:-librarian:build}"
export LIBRARIAN_DIREGAPIC_IMAGE_AND_TAG="${3:-librarian-diregapic:build}"


echo "CONVERTER_IMAGE_AND_TAG: ${CONVERTER_IMAGE_AND_TAG}"
echo "LIBRARIAN_STD_IMAGE_AND_TAG: ${LIBRARIAN_STD_IMAGE_AND_TAG}"
echo "LIBRARIAN_DIREGAPIC_IMAGE_AND_TAG: ${LIBRARIAN_DIREGAPIC_IMAGE_AND_TAG}"

echo "\n\n*** Dockerizing converter"
docker build -t ${CONVERTER_IMAGE_AND_TAG} "https://github.com/googleapis/disco-to-proto3-converter.git#main"

echo "\n\n*** Dockerizing standard Librarian"
docker build -t ${LIBRARIAN_STD_IMAGE_AND_TAG} .

echo "\n\n*** Dockerizing DIREGAPIC Librarian"
docker build \
  --build-arg CONVERTER_IMAGE_AND_TAG=${CONVERTER_IMAGE_AND_TAG} \
  --build-arg LIBRARIAN_STD_IMAGE_AND_TAG=${LIBRARIAN_STD_IMAGE_AND_TAG} \
  -t ${LIBRARIAN_DIREGAPIC_IMAGE_AND_TAG} \
  -f Dockerfile.DIREGAPIC \
  .


