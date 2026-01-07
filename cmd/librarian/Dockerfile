# Copyright 2026 Google LLC
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

# Multi-stage Dockerfile for language-specific librarian containers.
#
# Build:
#   docker build --target rust -t librarian:rust -f cmd/librarian/Dockerfile .
#   docker build --target python -t librarian:python -f cmd/librarian/Dockerfile .
#
# Usage:
#   docker run --rm -v "$(pwd)":/workspace -w /workspace librarian:rust generate
#   docker run --rm -v "$(pwd)":/workspace -w /workspace librarian:python generate

# ============== Build Stage ==============
FROM golang:1.25.5 AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build ./cmd/librarian

# ============== Base Stage ==============
FROM debian:bookworm-slim AS base

WORKDIR /app

RUN apt-get update && apt-get install -y \
    unzip curl git ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Install protoc v29.3
RUN curl -fsSL -o /tmp/protoc.zip \
    https://github.com/protocolbuffers/protobuf/releases/download/v29.3/protoc-29.3-linux-x86_64.zip \
    && cd /usr/local && unzip -o /tmp/protoc.zip && rm /tmp/protoc.zip

COPY --from=build /src/librarian .
ENTRYPOINT ["/app/librarian"]

# ============== Rust Stage ==============
FROM base AS rust

# Install Rust toolchain
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
ENV PATH="/root/.cargo/bin:${PATH}"

# Install taplo
RUN curl -fsSL https://github.com/tamasfe/taplo/releases/download/0.10.0/taplo-linux-x86_64.gz \
    | gunzip -c - > /usr/local/bin/taplo && chmod +x /usr/local/bin/taplo

# ============== Python Stage ==============
FROM base AS python

RUN apt-get update && apt-get install -y \
    python3 python3-pip python3-venv \
    && rm -rf /var/lib/apt/lists/*

# Install pandoc
RUN curl -fsSL -o /tmp/pandoc.tar.gz \
    https://github.com/jgm/pandoc/releases/download/3.8.2/pandoc-3.8.2-linux-amd64.tar.gz \
    && tar -xvf /tmp/pandoc.tar.gz -C /usr/local --strip-components=1 \
    && rm /tmp/pandoc.tar.gz

# Install Python dependencies
RUN pip install --no-cache-dir --break-system-packages \
    gapic-generator==1.28.3 \
    nox \
    gcp-synthtool@git+https://github.com/googleapis/synthtool@5aa438a342707842d11fbbb302c6277fbf9e4655
