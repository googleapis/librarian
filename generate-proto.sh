#!/bin/sh

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

# Regenerates the code in internal/statepb from the protos in the proto/
# directory. (See proto/README.md for why we have those protos.)
# Requirements: protoc and the protoc-gen-go plugin are in the path.
# Install protoc-gen-go with:
#   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
# (and make sure your go/bin directory is in your path)

protoc -I=proto --go_out=internal/statepb --go_opt=paths=source_relative proto/*.proto
