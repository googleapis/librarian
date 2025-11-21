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
FROM marketplace.gcr.io/google/debian12:latest AS builder

# Set environment variables for tool versions for easy updates.
ENV GO_VERSION=1.25.3

# Install build dependencies.
RUN apt-get update && \
    apt-get install -y \
    build-essential \
    ca-certificates \
    curl \
    wget && \
    rm -rf /var/lib/apt/lists/*

# Install the specific Go version required for compatibility.
RUN wget https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz -O go.tar.gz && \
    tar -C /usr/local -xzf go.tar.gz && \
    rm go.tar.gz
ENV PATH=/usr/local/go/bin:$PATH
RUN go version

WORKDIR /src

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build ./cmd/automation

FROM golang:1.25.3

WORKDIR /app

COPY --from=builder /src/automation .
ENTRYPOINT [ "/app/automation" ]
