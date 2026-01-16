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

# This stage builds the librariangen binary using the MOSS-compliant base image.
FROM marketplace.gcr.io/google/debian12@sha256:326ccf35aa72f7cc8cbd25b69e819a81c5fd9ed675c6c9ffb51f3214a64f23cf AS builder

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      jq \
    && apt-get clean && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY infra/imagebuilders/container/run.sh /app/run.sh

ENTRYPOINT [ "/app/run.sh" ]
