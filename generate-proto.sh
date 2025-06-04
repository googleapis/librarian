#!/bin/sh

# Regenerates the code in internal/statepb from the protos in the proto/
# directory. (See proto/README.md for why we have those protos.)
# Requirements: protoc and the protoc-gen-go plugin are in the path.
# Install protoc-gen-go with:
#   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
# (and make sure your go/bin directory is in your path)

protoc -I=proto --go_out=internal/statepb --go_opt=paths=source_relative proto/*.proto
