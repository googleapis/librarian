# syntax=docker/dockerfile:1
# The BuildKit syntax directive enables BuildKit features such as cache mounts
# for improved build performance.

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

# Prerequisites:
#   To build this image, you must have access to marketplace.gcr.io. Run the
#   following commands to authenticate and configure Docker to use your gcloud
#   credentials:
#
#   gcloud auth login
#   gcloud auth configure-docker marketplace.gcr.io
#
# Build:
#   docker build --target rust -t librarian:rust -f cmd/librarian/Dockerfile .
#   docker build --target python -t librarian:python -f cmd/librarian/Dockerfile .
#
# Usage:
#   docker run --rm -v "$(pwd)":/workspace -w /workspace librarian:rust generate
#   docker run --rm -v "$(pwd)":/workspace -w /workspace librarian:python generate
#

# Base/Build dependency versions
ARG TARGETARCH
ARG GO_VERSION=1.25.7
ARG PROTOC_VERSION=29.3
ARG PROTOC_CHECKSUM=f149c4a69910c8d241d27869d2cdf6791187a7a9434ad0f392b4a61e1697c3844e60bce634bd7147cb4cb7f2ad2530fc933be4976f67ac75d9ad78bb8b412f24

# Rust dependency versions
ARG RUST_VERSION=1.92
ARG TAPLO_CLI_VERSION=0.10.0
ARG CARGO_WORKSPACES_VERSION=0.4.0
ARG CARGO_SEMVER_CHECKS_VERSION=0.46.0

# Python dependency versions
ARG PYTHON_GAPIC_VERSION=1.30.10
ARG NOX_VERSION=2025.11.12
ARG RUFF_VERSION=0.14.14
ARG ISORT_VERSION=5.11.0
ARG BLACK_VERSION=23.7.0
ARG SYNTHTOOL_COMMIT=afd7725f32d522a95e964884e0fba6d0d6b4cb6a
ARG PANDOC_VERSION=3.8.2
ARG PANDOC_CHECKSUM=f5e90628b44a801ecdf812236291871607db8fb2fba0e99911a6c497b406c64a8940fd2a776d482ece7a9dac10d0228378d78da61091262b69e7276456495f5c

# Go dependency versions
ARG GO_PROTOC_PLUGIN_VERSION=1.35.2
ARG GO_PROTOC_GRPC_PLUGIN_VERSION=1.3.0
ARG GO_GAPIC_VERSION=0.57.0
ARG GO_GOIMPORTS_VERSION=latest


# ============== Build Stage ==============
# Use the MOSS-compliant base image.
FROM marketplace.gcr.io/google/debian12@sha256:326ccf35aa72f7cc8cbd25b69e819a81c5fd9ed675c6c9ffb51f3214a64f23cf AS build
ARG GO_VERSION
ARG TARGETARCH
ENV GO_VERSION=$GO_VERSION

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    build-essential \
    ca-certificates \
    curl \
    git \
    wget && \
    rm -rf /var/lib/apt/lists/*

RUN wget -q https://go.dev/dl/go${GO_VERSION}.linux-${TARGETARCH}.tar.gz -O go.tar.gz && \
    tar -C /usr/local -xzf go.tar.gz && \
    rm go.tar.gz
ENV PATH=/usr/local/go/bin:$PATH

WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -o /go/bin/librarian ./cmd/librarian

# ============== Base Stage ==============
FROM debian:bookworm-slim AS base
ARG PROTOC_VERSION
ARG PROTOC_CHECKSUM

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    unzip curl git ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Install protoc
RUN curl -fsSL -o /tmp/protoc.zip \
        https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip && \
    echo "${PROTOC_CHECKSUM} /tmp/protoc.zip" | sha512sum -c - && \
    cd /usr/local && unzip -o /tmp/protoc.zip && rm /tmp/protoc.zip

COPY --from=build /go/bin/librarian .
ENTRYPOINT ["/app/librarian"]

# ============== Rust Stage ==============
FROM base AS rust
ARG TAPLO_CLI_VERSION
ARG CARGO_WORKSPACES_VERSION
ARG CARGO_SEMVER_CHECKS_VERSION

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Install Rust toolchain
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
ENV PATH="/root/.cargo/bin:${PATH}"

# Install Rust and components
RUN rustup install ${RUST_VERSION} && \
    rustup default ${RUST_VERSION} && \
    rustup component add clippy rustfmt

# Install Rust tools
RUN cargo install --locked taplo-cli@${TAPLO_CLI_VERSION} && \
    cargo install --locked cargo-workspaces@${CARGO_WORKSPACES_VERSION} && \
    cargo install --locked cargo-semver-checks@${CARGO_SEMVER_CHECKS_VERSION} && \
    cargo install --locked cargo-minimal-versions && \
    cargo install --locked cargo-hack

# TODO: Reinstate cargo clean?

# Make sure the tools are executable by any user (not just root).
# librarianops executes the docker image as the current user.
RUN chmod a+wx /root
RUN chmod -R a+w /root/.cargo
ENV HOME=/root

# ============== Python Stage ==============
FROM base AS python
ARG PYTHON_GAPIC_VERSION
ARG NOX_VERSION
ARG RUFF_VERSION
ARG ISORT_VERSION
ARG BLACK_VERSION
ARG SYNTHTOOL_COMMIT
ARG PANDOC_VERSION
ARG PANDOC_CHECKSUM

RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 python3-pip python3-venv \
    && rm -rf /var/lib/apt/lists/*

# Install pandoc
RUN curl -fsSL -o /tmp/pandoc.tar.gz \
        https://github.com/jgm/pandoc/releases/download/${PANDOC_VERSION}/pandoc-${PANDOC_VERSION}-linux-amd64.tar.gz && \
    echo "${PANDOC_CHECKSUM} /tmp/pandoc.tar.gz" | sha512sum -c - && \
    tar -xvf /tmp/pandoc.tar.gz -C /usr/local --strip-components=1 && \
    rm /tmp/pandoc.tar.gz

# Install Python dependencies
RUN --mount=type=cache,target=/root/.cache/pip \
    pip install --no-cache-dir --break-system-packages \
    gapic-generator==${PYTHON_GAPIC_VERSION} \
    nox==${NOX_VERSION} \
    ruff==${RUFF_VERSION} \
    isort==${ISORT_VERSION} \
    black[jupyter]==${BLACK_VERSION} \
    gcp-synthtool@git+https://github.com/googleapis/synthtool@${SYNTHTOOL_COMMIT}

# ============== Go Stage ==============
FROM base AS go
ARG GO_VERSION
ARG TARGETARCH
ARG GO_PROTOC_PLUGIN_VERSION
ARG GO_PROTOC_GRPC_PLUGIN_VERSION
ARG GO_GAPIC_VERSION
ARG GO_GOIMPORTS_VERSION

RUN apt-get update && apt-get install -y --no-install-recommends \
    wget \
    && rm -rf /var/lib/apt/lists/*

RUN wget -q https://go.dev/dl/go${GO_VERSION}.linux-${TARGETARCH}.tar.gz -O go.tar.gz && \
    tar -C /usr/local -xzf go.tar.gz && \
    rm go.tar.gz
ENV PATH=/root/go/bin:/usr/local/go/bin:$PATH

# Install required Go tools for protoc and the post-processor.
RUN (export GOTOOLCHAIN='auto' && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v${GO_PROTOC_PLUGIN_VERSION} && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v${GO_PROTOC_GRPC_PLUGIN_VERSION} && \
    go install github.com/googleapis/gapic-generator-go/cmd/protoc-gen-go_gapic@v${GO_GAPIC_VERSION} && \
    go install golang.org/x/tools/cmd/goimports@${GO_GOIMPORTS_VERSION})

# Make sure the tools are executable by any user (not just root).
# librarianops executes the docker image as the current user.
RUN chmod a+rx /root/go/bin/*
RUN chmod a+x /root
RUN chmod a+x /root/go
RUN chmod a+x /root/go/bin
