#!/bin/bash
# Copyright 2024 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

install_go() {
    GO_VERSION="1.23.4"
    INSTALL_DIR="/usr/local"
    SHA256="6924efde5de86fe277676e929dc9917d466efa02fb934197bc2eba35d5680971"
    GO_ARCHIVE="go$GO_VERSION.linux-amd64.tar.gz"
    
    wget -q "https://go.dev/dl/$GO_ARCHIVE"
    echo "$SHA256 $GO_ARCHIVE" | sha256sum --check 
    
    tar -C $INSTALL_DIR -xzf $GO_ARCHIVE
    rm $GO_ARCHIVE
    echo "export PATH=$PATH:$INSTALL_DIR/go/bin" >> ~/.bashrc
    source ~/.bashrc
}

reduce_noise() {
    CI=true
    export CI
}

install_go
reduce_noise

WORK_ROOT=$KOKORO_ARTIFACTS_DIR/cli-work-root
mkdir $WORK_ROOT

gcloud auth configure-docker us-central1-docker.pkg.dev
cd github/generator
go run ./cmd/generator update-repo -language=dotnet -work-root=$WORK_ROOT
