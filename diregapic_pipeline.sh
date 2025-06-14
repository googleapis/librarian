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


# This is the "inner" script that coordinates the various steps in the DIREGAPIC pipeline. Assumes:
# - the Discovery file to start from is in the directory /app/discoveries/
# - the disco-to-proto3-converter configuration file is in /app/diregapic/
# The output of the converter is placed in /app/output
#
# USAGE:
#   diregapic_pipeline.sh API_NAME

if [[ "${BASH_SOURCE[0]}" != "$0" ]]; then
  echo "ERROR: This script should be executed directly, not sourced."
  return -1
fi

cd /app

### Validate inputs
API_NAME="$1"
[[ -n "${API_NAME}" ]] || { echo "error: This script requires an API name" ; exit -2 ; }
shift

[ $(ls -1 discoveries/*.json 2>/dev/null | wc -l) -eq 1 ] || {
  echo "Error: Expected exactly one Discovery .json file in /app/discoveries/. Got:" >&2
  ls -la discoveries/*json >&2
  exit -2
}
DISCOVERY_FILE=$(realpath discoveries/*.json )

[ $(ls -1 diregapic/*.json 2>/dev/null | wc -l) -eq 1 ] || {
  echo "Error: Expected exactly one converter configuration file in /app/diregapic/. Got:" >&2
  ls -la diregapic/*json >&2
  exit -2
}
CONFIGURATION_FILE=$(realpath diregapic/*.json )

OUTPUT_PROTO_FILE="$(realpath output/${API_NAME}.proto)"
OUTPUT_CONFIGURATION_FILE="$(realpath output/converter_config.json)"

### Create synthetic protos

# Ensure Discovery file is normalized so it's easier to track changes and errors.
python3 normalize_discovery.py

java -jar disco-to-proto3-converter.jar \
  --discovery_doc_path=${DISCOVERY_FILE} \
  --input_config_path=${CONFIGURATION_FILE} \
  --output_file_path=${OUTPUT_PROTO_FILE} \
  --output_config_path=${OUTPUT_CONFIGURATION_FILE} \
  --enums_as_strings=True

# TODO: Pass the synthetic proto to the rest of the librarian pipeline. Maybe
# something like this:
#   librarian "$@"
# (but including the synthetic proto) so that additional Librarian args can be
# passed to this script and they will passed to the librarian executable.
