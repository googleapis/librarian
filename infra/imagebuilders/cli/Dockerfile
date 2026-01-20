# Copyright 2024 Google LLC
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

# Set environment variables for tool versions for easy updates.
ENV GO_VERSION=1.25.3

# Install build dependencies.
RUN apt-get update && \
    apt-get install -y \
    build-essential \
    ca-certificates \
    curl \
    git \
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
# Renaming the binary to librarian to match the old binary name and avoid
# breaking existing Cloud Build steps that reference the binary by name.
RUN CGO_ENABLED=0 GOOS=linux go build -o librarian ./cmd/legacylibrarian

# Using docker:dind so we can run docker from the CLI,
# while in Docker. Note that for this to work, *this*
# docker image should be run with
#  -v /var/run/docker.sock:/var/run/docker.sock
FROM marketplace.gcr.io/google/debian12@sha256:326ccf35aa72f7cc8cbd25b69e819a81c5fd9ed675c6c9ffb51f3214a64f23cf
WORKDIR /app

# From https://docs.docker.com/engine/install/debian/

RUN apt update
RUN apt-get install -y unzip \
  gnupg \
  apt-transport-https \
  ca-certificates \
  curl \
  && rm -rf /var/lib/apt/lists/*

# Add Docker's official GPG key
RUN install -m 0755 -d /etc/apt/keyrings
RUN curl -fsSL https://download.docker.com/linux/debian/gpg -o /etc/apt/keyrings/docker.asc
RUN chmod a+r /etc/apt/keyrings/docker.asc

# Add the repository to Apt sources:
RUN echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian \
    $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
      tee /etc/apt/sources.list.d/docker.list > /dev/null

# Install Docker and pin version
# Cloud Build runs Docker Engine v20.10.24, which supports a maximum API version of v1.41.
# https://docs.cloud.google.com/build/docs/overview#docker
# Docker client v29.0.0 and newer require API version v1.44 or later.
# https://docs.docker.com/engine/release-notes/29/#2900
RUN apt update && \
    apt-get -y install \
    docker-ce=5:27.1.1-1~debian.12~bookworm \
    docker-ce-cli=5:27.1.1-1~debian.12~bookworm \
    docker-ce-rootless-extras=5:27.1.1-1~debian.12~bookworm && \
    rm -rf /var/lib/apt/lists/*

# Add the Google Cloud SDK distribution URI as a package source
RUN echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" \
    > /etc/apt/sources.list.d/google-cloud-sdk.list && \
    curl -fsSL https://packages.cloud.google.com/apt/doc/apt-key.gpg | \
    gpg --dearmor -o /usr/share/keyrings/cloud.google.gpg

# Install the gcloud CLI
RUN apt-get update && apt-get install -y google-cloud-sdk

COPY --from=builder /src/librarian .
ENTRYPOINT ["/app/librarian"]
