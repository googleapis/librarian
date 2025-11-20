# Copyright 2025 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Start with a Go base image
FROM golang:1.25.3 AS builder

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY ./internal/legacylibrarian/legacyintegration/testdata/e2e_func.go .

RUN go build -o e2e_func .

FROM alpine:latest

WORKDIR /app

# Copy the built executable from the builder stage
COPY --from=builder /app/e2e_func .

ENTRYPOINT ["/app/e2e_func"]
