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

# How to build these images, and when to update the versions below:
# doc/language-images.md.

# Each *_CHECKSUM is the checksum the publisher lists for that download. For
# downloads without one (protoc, pandoc, Python and the Node generator), it is
# the checksum of the official download, matched against the GitHub release
# asset digest or the release signature where one exists.

# Base/Build dependency versions
# Debian 12, pinned by digest so rebuilds are reproducible. This is the
# bookworm-20261104-slim tag, which is not the newest build.
ARG BASE_IMAGE=debian:bookworm-slim@sha256:734a72b3d6142a71993b4bc580ca01d8ca687f77a2c422d96a961d3376b179bc
# Compiles librarian. Matches the go directive in librarian's go.mod.
ARG GO_VERSION=1.26.1
ARG GO_CHECKSUM=031f088e5d955bab8657ede27ad4e3bc5b7c1ba281f05f245bcc304f327c987a
# Matches tools.protoc in the librarian.yaml of google-cloud-rust,
# google-cloud-ruby and google-cloud-swift. Librarian uses the protoc from a
# repository's librarian.yaml when it sets one, so this is a fallback.
ARG PROTOC_VERSION=33.2
ARG PROTOC_CHECKSUM=8c0a8577311720cc83488f813aeb5046d69cb9824cbf39cb7447e0c5132d677952d5bb7c1130d9df381835eee7352828878d409873ebb407d7a4649ea2a4aee5

# Rust dependency versions
# Latest rustup release when pinned.
ARG RUSTUP_VERSION=1.29.3
ARG RUSTUP_CHECKSUM=b1c8cb14e30889e7e2999e6b69e62d9e9b9ffc0f4df8180564d306d9680e6c55
# google-cloud-rust needs Rust 1.99 or later for its CI formatting checks.
# 1.100.0 matches the channel (1.100) in its rust-toolchain.toml.
ARG RUST_VERSION=1.100.0
# Match tools.cargo in google-cloud-rust's librarian.yaml.
ARG TAPLO_CLI_VERSION=0.10.0
ARG CARGO_WORKSPACES_VERSION=0.4.0
ARG CARGO_SEMVER_CHECKS_VERSION=0.50.0
ARG CARGO_MINIMAL_VERSIONS_VERSION=0.1.37
ARG CARGO_HACK_VERSION=0.6.44

# Python dependency versions
# Generating google-cloud-python needs Python 3.10 or later. 3.10.12 is the
# python3 in Ubuntu 22.04, which is not the latest 3.10 patch.
ARG PYTHON_VERSION=3.10.12
ARG PYTHON_CHECKSUM=afb74bf19130e7a47d10312c8f5e784f24e0527981eab68e20546cfb865830b8
# Shared libraries the compiled Python needs at run time.
ARG PYTHON_RUNTIME_LIBRARIES="libbz2-1.0 libffi8 liblzma5 libncursesw6 libreadline8 libsqlite3-0 libssl3 libuuid1 zlib1g"
# Matches tools.protoc in google-cloud-python's librarian.yaml.
ARG PYTHON_PROTOC_VERSION=33.5
ARG PYTHON_PROTOC_CHECKSUM=eb842a2259a90a402207cae3a9261dc9501f43a535b3b3d786ecc496e117f499cf373fe41181fd9047035f7ed6c2dee9a55f7f0a7afa1c5d7cb5be3e672bf527
# pandoc 3.8.2 is the version the Python generation setup requires.
# pypandoc-binary 1.16.2, which `librarian install` installs, bundles pandoc
# 3.8.2.1. Newer pandoc releases (3.10 and later) reject multiline links in
# the generated docs.
ARG PANDOC_VERSION=3.8.2
ARG PANDOC_CHECKSUM=f5e90628b44a801ecdf812236291871607db8fb2fba0e99911a6c497b406c64a8940fd2a776d482ece7a9dac10d0228378d78da61091262b69e7276456495f5c

# Go dependency versions
# Matches the go directive in google-cloud-go's go.mod. Every image has Go.
ARG GO_RUNTIME_VERSION=1.26.0
ARG GO_RUNTIME_CHECKSUM=aac1b08a0fb0c4e0a7c1555beb7b59180b05dfc5a3d62e40e9de90cd42f88235
# protoc and the generator plugins are from earlier images. `librarian
# install` installs the versions in google-cloud-go's librarian.yaml.
ARG GO_PROTOC_VERSION=25.7
ARG GO_PROTOC_CHECKSUM=ac571249613458e4b9598a8f2a9b3ab9e56ed04d35f610f4181c65da4ee9bfd6fe44b5b0fd0a6d5b53ca4657d443c3bd457c3ef6164df8b8b4bc537012205aec
ARG GO_PROTOC_PLUGIN_VERSION=1.35.2
ARG GO_PROTOC_GRPC_PLUGIN_VERSION=1.3.0
ARG GO_GAPIC_VERSION=0.59.0
# Matches goimports in google-cloud-go's librarian.yaml.
ARG GO_GOIMPORTS_VERSION=0.44.0

# NodeJS dependency versions
# google-cloud-node needs Node 22, which its CI uses. 22.23.5 is the latest
# Node 22 release when pinned.
ARG NODEJS_NODE_VERSION=22.23.5
ARG NODEJS_NODE_CHECKSUM=c54118104acc524633b5abe106bbf44a53126ec8b4e5fa105cb92b5d7de8a1a1
# Matches google-cloud-node's packageManager (pnpm@10.0.0). Otherwise pnpm
# downloads that version itself.
ARG NODEJS_PNPM_VERSION=10.0.0
# Generator tools from earlier images. `librarian install` installs the
# versions in google-cloud-node's librarian.yaml.
ARG NODEJS_GAPIC_GENERATOR_VERSION=gapic-generator-v4.12.1
ARG NODEJS_GAPIC_GENERATOR_CHECKSUM=477e4a9b3442457667846bd5c629d1c059e7caf2d5a3ebce95da626a2d42ea80
ARG NODEJS_GAPIC_NODE_PROCESSING_VERSION=0.1.8
ARG NODEJS_GAPIC_TOOLS_VERSION=1.0.6
ARG NODEJS_PROTOBUFJS_CLI_VERSION=1.2.0
ARG NODEJS_PROTOBUFJS_VERSION=7.5.8

# Java dependency versions
# Generating google-cloud-java needs JDK 17, which its CI uses, and Maven
# 3.9.8. Temurin 17.0.7+7 is the JDK the SDK team generates it with, which is
# not the latest 17 update.
ARG JAVA_VERSION=17.0.7
ARG JAVA_BUILD=7
ARG JAVA_CHECKSUM=e9458b38e97358850902c2936a1bb5f35f6cffc59da9fcd28c63eab8dbbfbc3b
ARG JAVA_MAVEN_VERSION=3.9.8
ARG JAVA_MAVEN_CHECKSUM=7d171def9b85846bf757a2cec94b7529371068a0670df14682447224e57983528e97a6d1b850327e4ca02b139abaab7fcb93c4315119e6f0ffb3f0cbc0d0b9a2

# PHP dependency versions
# google-cloud-php needs PHP 8.2 or later, as its composer.json says. 8.4.27
# comes from the initial runtime list for these images and is kept without a
# bump, so it is not the latest 8.4 patch.
ARG PHP_VERSION=8.4.27
ARG PHP_CHECKSUM=dcf93923b0e38e0d5252b25b796520b5a47761a08b3488b6d0b5d1115764d3f0
# Shared libraries the compiled PHP needs at run time.
ARG PHP_RUNTIME_LIBRARIES="libcurl4 libicu72 libonig5 libsqlite3-0 libssl3 libxml2 libzip4 zlib1g"
# The last Composer 2.9 release. It comes from the initial runtime list for
# these images and is kept without a bump, so it is older than the newest
# Composer.
ARG PHP_COMPOSER_VERSION=2.9.8
ARG PHP_COMPOSER_CHECKSUM=59b2c50e10cafa0d8efc19ede9a326d782f096c674a26baf98cf042ce23de890
# Last pnpm 7 release. Librarian's PHP formatting finds @prettier/plugin-php
# in pnpm 7's global directory layout, and google-cloud-php CI uses pnpm 7.
ARG PHP_PNPM_VERSION=7.33.7

# Ruby dependency versions
# google-cloud-ruby needs Ruby 3.2 or later, and its CI tests Ruby 3.2 to
# 4.0. 3.4.9 is the latest Ruby 3.4 release when pinned.
ARG RUBY_VERSION=3.4.9
ARG RUBY_CHECKSUM=4231c54072601a171faed1699f105985e9971c94cd382b78feb4eb44eec2dd1a
# Headers that Ruby and native gem extensions build against.
ARG RUBY_DEV_LIBRARIES="libffi-dev libssl-dev libyaml-dev zlib1g-dev"

# Swift dependency versions
# google-cloud-swift needs Swift 6.2 or later, as its Package.swift says, and
# its CI uses Swift 6.4. 6.4.2 comes from the initial runtime list for these
# images and is kept without a bump, so it is not the latest 6.4 patch.
ARG SWIFT_VERSION=6.4.2
# Swift 6.x Release Signing Key, from https://www.swift.org/keys/.
ARG SWIFT_SIGNING_FINGERPRINT=52BB7E3DE28A71BE22EC05FFEF80A866B47A981F

# ============== Build Stage ==============
# Use the MOSS-compliant base image, pinned by digest. This is the
# 2025-11-18 build of the latest tag, which is not the newest build. It only
# compiles librarian, so its OS is not in the language images.
FROM --platform=linux/amd64 marketplace.gcr.io/google/debian12@sha256:326ccf35aa72f7cc8cbd25b69e819a81c5fd9ed675c6c9ffb51f3214a64f23cf AS build
ARG GO_VERSION
ARG GO_CHECKSUM
ENV GO_VERSION=$GO_VERSION

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    build-essential \
    ca-certificates \
    curl \
    git \
    wget && \
    rm -rf /var/lib/apt/lists/*

RUN wget -q https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz -O go.tar.gz && \
    echo "${GO_CHECKSUM} go.tar.gz" | sha256sum -c - && \
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
    CGO_ENABLED=0 go build -buildvcs=false -o /go/bin/librarian ./cmd/librarian

# ============== Runtime Builder Stage ==============
# Runtimes are downloaded or compiled in stages that do not depend on
# librarian, so changes to librarian or the language tools do not rebuild
# them. The language stages copy the finished runtimes from these stages.
FROM --platform=linux/amd64 ${BASE_IMAGE} AS runtime-builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential ca-certificates curl pkg-config xz-utils \
    && rm -rf /var/lib/apt/lists/*

# ============== Go Runtime Stage ==============
FROM runtime-builder AS go-runtime
ARG GO_RUNTIME_VERSION
ARG GO_RUNTIME_CHECKSUM

# Install Go from the official binaries.
RUN curl -fsSL --retry 5 --retry-delay 15 -o /tmp/go.tar.gz \
        https://go.dev/dl/go${GO_RUNTIME_VERSION}.linux-amd64.tar.gz && \
    echo "${GO_RUNTIME_CHECKSUM} /tmp/go.tar.gz" | sha256sum -c - && \
    tar -C /usr/local -xzf /tmp/go.tar.gz && \
    rm /tmp/go.tar.gz

# ============== Base Stage ==============
FROM --platform=linux/amd64 ${BASE_IMAGE} AS base
ARG PROTOC_VERSION
ARG PROTOC_CHECKSUM

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    unzip curl git ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Jobs run the images as any UID, often one with no home directory, on a
# checkout owned by another user. Give every user a writable home, where
# Librarian and the tools keep their caches, and let git use any checkout.
# Stages that install tools as root at build time use HOME=/root for that.
ENV HOME=/home/librarian
RUN mkdir -m 777 -p ${HOME} && \
    git config --system --add safe.directory '*'

# Install protoc
RUN curl -fsSL --retry 5 --retry-delay 15 -o /tmp/protoc.zip \
        https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip && \
    echo "${PROTOC_CHECKSUM} /tmp/protoc.zip" | sha512sum -c - && \
    cd /usr/local && unzip -o /tmp/protoc.zip && rm /tmp/protoc.zip

# Every image has Go, so CI can `go install` or `go run` the librarian
# version that a repository pins in librarian.yaml.
COPY --from=go-runtime /usr/local/go /usr/local/go
ENV PATH=/usr/local/go/bin:$PATH
RUN go version

COPY --from=build /go/bin/librarian .
ENTRYPOINT ["/app/librarian"]

# ============== Rust Stage ==============
FROM base AS rust
ARG RUSTUP_VERSION
ARG RUSTUP_CHECKSUM
ARG RUST_VERSION
ARG TAPLO_CLI_VERSION
ARG CARGO_WORKSPACES_VERSION
ARG CARGO_SEMVER_CHECKS_VERSION
ARG CARGO_MINIMAL_VERSIONS_VERSION
ARG CARGO_HACK_VERSION
# The toolchain and Cargo tools stay in /root for every user and HOME.
ENV RUSTUP_HOME=/root/.rustup
ENV CARGO_HOME=/root/.cargo

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Install Rust toolchain
RUN curl -fsSL --retry 5 --retry-delay 15 -o /tmp/rustup-init \
        https://static.rust-lang.org/rustup/archive/${RUSTUP_VERSION}/x86_64-unknown-linux-gnu/rustup-init && \
    echo "${RUSTUP_CHECKSUM} /tmp/rustup-init" | sha256sum -c - && \
    chmod +x /tmp/rustup-init && \
    /tmp/rustup-init -y --default-toolchain none --no-modify-path && \
    rm /tmp/rustup-init
ENV PATH="/root/.cargo/bin:${PATH}"

# Install Rust and components
RUN rustup install ${RUST_VERSION} && \
    rustup default ${RUST_VERSION} && \
    rustup component add clippy rustfmt

# Install Rust tools. Make sure the tools are usable by any user (not just
# root), because librarianops executes the docker image as the current user.
# The chmod runs in the same layer, so the files are not copied again.
RUN cargo install --locked taplo-cli@${TAPLO_CLI_VERSION} && \
    cargo install --locked cargo-workspaces@${CARGO_WORKSPACES_VERSION} && \
    cargo install --locked cargo-semver-checks@${CARGO_SEMVER_CHECKS_VERSION} && \
    cargo install --locked cargo-minimal-versions@${CARGO_MINIMAL_VERSIONS_VERSION} && \
    cargo install --locked cargo-hack@${CARGO_HACK_VERSION} && \
    chmod a+x /root && \
    chmod -R a+rwX /root/.cargo
# rust-toolchain.toml files use the x.y channel name, which rustup treats as a
# different toolchain than x.y.z. Link it so it resolves without a download.
RUN ln -s ${RUST_VERSION}-x86_64-unknown-linux-gnu /root/.rustup/toolchains/${RUST_VERSION%.*}-x86_64-unknown-linux-gnu

# Fail the build if a runtime the image promises is missing.
RUN --mount=type=tmpfs,target=/home/librarian \
    cargo --version && rustc --version

# ============== Python Runtime Stage ==============
# Python has no official Linux binaries, so it is compiled from the official
# source.
FROM runtime-builder AS python-runtime
ARG PYTHON_VERSION
ARG PYTHON_CHECKSUM

RUN apt-get update && apt-get install -y --no-install-recommends \
    libbz2-dev libffi-dev liblzma-dev libncurses-dev libreadline-dev \
    libsqlite3-dev libssl-dev uuid-dev zlib1g-dev \
    && rm -rf /var/lib/apt/lists/*

# The test suite and static libraries are not needed at run time, so they are
# removed in the same layer as `make install`.
RUN curl -fsSL --retry 5 --retry-delay 15 -o /tmp/python.tar.xz \
        https://www.python.org/ftp/python/${PYTHON_VERSION}/Python-${PYTHON_VERSION}.tar.xz && \
    echo "${PYTHON_CHECKSUM} /tmp/python.tar.xz" | sha256sum -c - && \
    mkdir /tmp/python && \
    tar -xJf /tmp/python.tar.xz -C /tmp/python --strip-components=1 && \
    cd /tmp/python && \
    ./configure --prefix=/opt/python --with-ensurepip=install && \
    make -j"$(nproc)" && \
    make install && \
    rm -rf /opt/python/lib/python3*/test /opt/python/lib/libpython3*.a \
        /opt/python/lib/python3*/config-3*/libpython3*.a && \
    ln -s python3 /opt/python/bin/python && \
    ln -s pip3 /opt/python/bin/pip && \
    cd / && rm -rf /tmp/python /tmp/python.tar.xz

# ============== PHP Runtime Stage ==============
# PHP has no official Linux binaries, so it is compiled from the official
# source with the extensions Composer and the PHP generator need. Only the CLI
# is built, because nothing uses the CGI or phpdbg programs.
FROM runtime-builder AS php-runtime
ARG PHP_VERSION
ARG PHP_CHECKSUM

RUN apt-get update && apt-get install -y --no-install-recommends \
    libcurl4-openssl-dev libicu-dev libonig-dev libsqlite3-dev libssl-dev \
    libxml2-dev libzip-dev zlib1g-dev \
    && rm -rf /var/lib/apt/lists/*

RUN curl -fsSL --retry 5 --retry-delay 15 -o /tmp/php.tar.xz \
        https://www.php.net/distributions/php-${PHP_VERSION}.tar.xz && \
    echo "${PHP_CHECKSUM} /tmp/php.tar.xz" | sha256sum -c - && \
    mkdir /tmp/php && \
    tar -xJf /tmp/php.tar.xz -C /tmp/php --strip-components=1 && \
    cd /tmp/php && \
    ./configure --prefix=/opt/php --with-openssl --with-curl --with-zlib \
        --with-zip --enable-bcmath --enable-intl --enable-mbstring \
        --disable-cgi --disable-phpdbg && \
    make -j"$(nproc)" && \
    make install && \
    strip --strip-debug /opt/php/bin/php && \
    cd / && rm -rf /tmp/php /tmp/php.tar.xz

# ============== Ruby Runtime Stage ==============
# Ruby has no official Linux binaries, so it is compiled from the official
# source, without the generated API documentation or debug symbols.
FROM runtime-builder AS ruby-runtime
ARG RUBY_VERSION
ARG RUBY_CHECKSUM
ARG RUBY_DEV_LIBRARIES

RUN apt-get update && apt-get install -y --no-install-recommends \
    ${RUBY_DEV_LIBRARIES} \
    && rm -rf /var/lib/apt/lists/*

RUN curl -fsSL --retry 5 --retry-delay 15 -o /tmp/ruby.tar.xz \
        https://cache.ruby-lang.org/pub/ruby/${RUBY_VERSION%.*}/ruby-${RUBY_VERSION}.tar.xz && \
    echo "${RUBY_CHECKSUM} /tmp/ruby.tar.xz" | sha256sum -c - && \
    mkdir /tmp/ruby && \
    tar -xJf /tmp/ruby.tar.xz -C /tmp/ruby --strip-components=1 && \
    cd /tmp/ruby && \
    ./configure --prefix=/opt/ruby --enable-shared --disable-install-doc && \
    make -j"$(nproc)" && \
    make install && \
    strip --strip-debug /opt/ruby/bin/ruby && \
    find /opt/ruby/lib -name '*.so*' -type f -exec strip --strip-debug {} + && \
    cd / && rm -rf /tmp/ruby /tmp/ruby.tar.xz

# ============== Swift Runtime Stage ==============
# Swift publishes a toolchain per Linux distribution, signed with the Swift
# release signing key. The Debian 12 build matches the base image.
FROM runtime-builder AS swift-runtime
ARG SWIFT_VERSION
ARG SWIFT_SIGNING_FINGERPRINT

RUN apt-get update && apt-get install -y --no-install-recommends \
    dirmngr gnupg \
    && rm -rf /var/lib/apt/lists/*

RUN curl -fsSL --retry 5 --retry-delay 15 -o /tmp/swift.tar.gz \
        https://download.swift.org/swift-${SWIFT_VERSION}-release/debian12/swift-${SWIFT_VERSION}-RELEASE/swift-${SWIFT_VERSION}-RELEASE-debian12.tar.gz && \
    curl -fsSL --retry 5 --retry-delay 15 -o /tmp/swift.tar.gz.sig \
        https://download.swift.org/swift-${SWIFT_VERSION}-release/debian12/swift-${SWIFT_VERSION}-RELEASE/swift-${SWIFT_VERSION}-RELEASE-debian12.tar.gz.sig && \
    export GNUPGHOME="$(mktemp -d)" && \
    gpg --batch --keyserver hkps://keyserver.ubuntu.com --recv-keys ${SWIFT_SIGNING_FINGERPRINT} && \
    gpg --batch --verify /tmp/swift.tar.gz.sig /tmp/swift.tar.gz && \
    mkdir -p /opt/swift && \
    tar -xzf /tmp/swift.tar.gz -C /opt/swift --strip-components=1 --no-same-owner && \
    rm -rf "${GNUPGHOME}" /tmp/swift.tar.gz /tmp/swift.tar.gz.sig

# ============== NodeJS Runtime Stage ==============
FROM runtime-builder AS nodejs-runtime
ARG NODEJS_NODE_VERSION
ARG NODEJS_NODE_CHECKSUM

# Install Node from the official binaries.
RUN curl -fsSL --retry 5 --retry-delay 15 -o /tmp/node.tar.gz \
        https://nodejs.org/dist/v${NODEJS_NODE_VERSION}/node-v${NODEJS_NODE_VERSION}-linux-x64.tar.gz && \
    echo "${NODEJS_NODE_CHECKSUM} /tmp/node.tar.gz" | sha256sum -c - && \
    mkdir -p /usr/local/node && \
    tar -xzf /tmp/node.tar.gz -C /usr/local/node --strip-components=1 --no-same-owner && \
    rm /tmp/node.tar.gz

# ============== Python Base Stage ==============
# Python with a virtual environment, shared by the python and php images.
FROM base AS python-base
ARG PYTHON_RUNTIME_LIBRARIES

RUN apt-get update && apt-get install -y --no-install-recommends \
    ${PYTHON_RUNTIME_LIBRARIES} \
    && rm -rf /var/lib/apt/lists/*
COPY --from=python-runtime /opt/python /opt/python
ENV PATH=/opt/python/bin:${PATH}

# Jobs run `librarian install` before generating, which pip installs the
# Python tools. They go into a virtual environment in /python-packages, so it
# is predictable where they are installed. Putting it first on PATH makes its
# python3 and pip the defaults. A virtual environment is used instead of
# PIP_TARGET, because pip with --target skips the console scripts of later
# installs when the bin directory already exists.
ENV VIRTUAL_ENV=/python-packages
ENV PATH=${VIRTUAL_ENV}/bin:${PATH}
ENV PIP_NO_CACHE_DIR=1

# Grant broad write permissions, so any UID running the container (like
# librarianops) can run `librarian install`.
RUN python3 -m venv ${VIRTUAL_ENV} && \
    chmod -R a+rwX ${VIRTUAL_ENV}

# ============== Python Stage ==============
FROM python-base AS python
ARG PYTHON_PROTOC_VERSION
ARG PYTHON_PROTOC_CHECKSUM
ARG PANDOC_VERSION
ARG PANDOC_CHECKSUM

# Install pandoc
RUN curl -fsSL --retry 5 --retry-delay 15 -o /tmp/pandoc.tar.gz \
        https://github.com/jgm/pandoc/releases/download/${PANDOC_VERSION}/pandoc-${PANDOC_VERSION}-linux-amd64.tar.gz && \
    echo "${PANDOC_CHECKSUM} /tmp/pandoc.tar.gz" | sha512sum -c - && \
    tar -xvf /tmp/pandoc.tar.gz -C /usr/local --strip-components=1 && \
    rm /tmp/pandoc.tar.gz

# Install Python-specific protoc version
RUN curl -fsSL --retry 5 --retry-delay 15 -o /tmp/protoc.zip \
        https://github.com/protocolbuffers/protobuf/releases/download/v${PYTHON_PROTOC_VERSION}/protoc-${PYTHON_PROTOC_VERSION}-linux-x86_64.zip && \
    echo "${PYTHON_PROTOC_CHECKSUM} /tmp/protoc.zip" | sha512sum -c - && \
    cd /usr/local && unzip -o /tmp/protoc.zip && rm /tmp/protoc.zip

# Librarian keeps installed tools and downloads in its cache directory.
ENV LIBRARIAN_CACHE=/python-packages/librarian-cache
RUN mkdir -p ${LIBRARIAN_CACHE} && \
    chmod a+rwX ${LIBRARIAN_CACHE}

# Fail the build if a runtime the image promises is missing.
RUN --mount=type=tmpfs,target=/home/librarian \
    python3 --version && pip --version && pandoc --version

# ============== PHP Stage ==============
FROM python-base AS php
ARG PHP_RUNTIME_LIBRARIES
ARG PHP_COMPOSER_VERSION
ARG PHP_COMPOSER_CHECKSUM
ARG PHP_PNPM_VERSION

# Node requires libatomic but doesn't install it.
RUN apt-get update && apt-get install -y --no-install-recommends \
    libatomic1 ${PHP_RUNTIME_LIBRARIES} \
    && rm -rf /var/lib/apt/lists/*
COPY --from=php-runtime /opt/php /opt/php
COPY --from=nodejs-runtime /usr/local/node /usr/local/node
ENV PATH=${PATH}:/opt/php/bin:/usr/local/node/bin

# Install Composer from the official release.
RUN curl -fsSL --retry 5 --retry-delay 15 -o /usr/local/bin/composer \
        https://getcomposer.org/download/${PHP_COMPOSER_VERSION}/composer.phar && \
    echo "${PHP_COMPOSER_CHECKSUM} /usr/local/bin/composer" | sha256sum -c - && \
    chmod 755 /usr/local/bin/composer

# Install pnpm
RUN HOME=/root npm install --global pnpm@${PHP_PNPM_VERSION}

# Librarian keeps installed tools and downloads in its cache directory, and
# Composer keeps its settings and cache in COMPOSER_HOME.
ENV LIBRARIAN_CACHE=/usr/local/php_tools/cache
ENV COMPOSER_HOME=/usr/local/php_tools/composer

# Grant broad write permissions, so any UID running the container (like
# librarianops) can run `librarian install php`.
RUN mkdir -p ${LIBRARIAN_CACHE} ${COMPOSER_HOME} && \
    chmod -R a+rwX /usr/local/php_tools

# Fail the build if a runtime the image promises is missing.
RUN --mount=type=tmpfs,target=/home/librarian --mount=type=tmpfs,target=/usr/local/php_tools/composer \
    php --version && composer --version && python3 --version && node --version && pnpm --version

# ============== Ruby Stage ==============
FROM base AS ruby
ARG RUBY_VERSION
ARG RUBY_DEV_LIBRARIES

# build-essential lets gems build native extensions against the Ruby headers.
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential ${RUBY_DEV_LIBRARIES} \
    && rm -rf /var/lib/apt/lists/*
COPY --from=ruby-runtime /opt/ruby /opt/ruby
ENV PATH=/opt/ruby/bin:${PATH}

# Librarian keeps installed tools and downloads in its cache directory. Gems
# are installed in GEM_HOME instead of the root-owned Ruby, which stays on
# GEM_PATH for its default gems.
ENV LIBRARIAN_CACHE=/usr/local/ruby_tools/cache
ENV GEM_HOME=/usr/local/ruby_tools/gems
ENV GEM_PATH=${GEM_HOME}:/opt/ruby/lib/ruby/gems/${RUBY_VERSION%.*}.0
ENV PATH=${GEM_HOME}/bin:${PATH}

# Grant broad write permissions, so any UID running the container (like
# librarianops) can run `librarian install ruby` and `bundle install`.
RUN mkdir -p ${LIBRARIAN_CACHE} ${GEM_HOME} && \
    chmod -R a+rwX /usr/local/ruby_tools

# Fail the build if a runtime the image promises is missing.
RUN --mount=type=tmpfs,target=/home/librarian \
    ruby --version && gem --version && bundle --version

# ============== Swift Stage ==============
FROM base AS swift

# Libraries and tools the Swift toolchain needs to compile and link.
RUN apt-get update && apt-get install -y --no-install-recommends \
    binutils gcc libc6-dev libcurl4-openssl-dev libedit-dev libicu-dev \
    libncurses-dev libsqlite3-dev libssl-dev libstdc++-12-dev libxml2-dev \
    pkg-config tzdata uuid-dev zlib1g-dev \
    && rm -rf /var/lib/apt/lists/*
COPY --from=swift-runtime /opt/swift /opt/swift
# Swift Build finds the toolchain only in its usr layout, so the toolchain
# keeps its usr directory.
ENV PATH=/opt/swift/usr/bin:${PATH}

# Librarian keeps installed tools and downloads in its cache directory.
ENV LIBRARIAN_CACHE=/usr/local/swift_tools/cache

# Grant broad write permissions, so any UID running the container (like
# librarianops) can run `librarian install swift`.
RUN mkdir -p ${LIBRARIAN_CACHE} && \
    chmod -R a+rwX /usr/local/swift_tools

# Fail the build if a runtime the image promises is missing.
RUN --mount=type=tmpfs,target=/home/librarian \
    swift --version && swift-format --version

# ============== Go Stage ==============
FROM base AS go
ARG GO_RUNTIME_VERSION
ARG GO_PROTOC_VERSION
ARG GO_PROTOC_CHECKSUM
ARG GO_PROTOC_PLUGIN_VERSION
ARG GO_PROTOC_GRPC_PLUGIN_VERSION
ARG GO_GAPIC_VERSION
ARG GO_GOIMPORTS_VERSION

ENV GO_VERSION=$GO_RUNTIME_VERSION

ENV PATH=/root/go/bin:$PATH

# Install required Go tools for protoc and the post-processor.
RUN (export GOTOOLCHAIN='auto' HOME=/root && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v${GO_PROTOC_PLUGIN_VERSION} && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v${GO_PROTOC_GRPC_PLUGIN_VERSION} && \
    go install github.com/googleapis/gapic-generator-go/cmd/protoc-gen-go_gapic@v${GO_GAPIC_VERSION} && \
    go install golang.org/x/tools/cmd/goimports@v${GO_GOIMPORTS_VERSION})

# Install Go-specific protoc version
RUN curl -fsSL --retry 5 --retry-delay 15 -o /tmp/protoc.zip \
        https://github.com/protocolbuffers/protobuf/releases/download/v${GO_PROTOC_VERSION}/protoc-${GO_PROTOC_VERSION}-linux-x86_64.zip && \
    echo "${GO_PROTOC_CHECKSUM} /tmp/protoc.zip" | sha512sum -c - && \
    cd /usr/local && unzip -o /tmp/protoc.zip && rm /tmp/protoc.zip

# Make sure the tools are executable by any user (not just root).
# librarianops executes the docker image as the current user.
RUN chmod a+rx /root/go/bin/* && \
    chmod a+x /root && \
    chmod a+x /root/go && \
    chmod a+x /root/go/bin

# Fail the build if a runtime the image promises is missing.
RUN --mount=type=tmpfs,target=/home/librarian \
    go version

# ============== Node Stage ==============
FROM base AS nodejs
ARG NODEJS_PNPM_VERSION
ARG NODEJS_GAPIC_GENERATOR_VERSION
ARG NODEJS_GAPIC_GENERATOR_CHECKSUM
ARG NODEJS_GAPIC_NODE_PROCESSING_VERSION
ARG NODEJS_GAPIC_TOOLS_VERSION
ARG NODEJS_PROTOBUFJS_CLI_VERSION
ARG NODEJS_PROTOBUFJS_VERSION

# Node requires libatomic but doesn't install it.
RUN apt-get update && apt-get install -y --no-install-recommends \
    libatomic1 \
    && rm -rf /var/lib/apt/lists/*

COPY --from=nodejs-runtime /usr/local/node /usr/local/node
ENV PATH=$PATH:/usr/local/node/bin

# Install pnpm
RUN HOME=/root npm install --global pnpm@${NODEJS_PNPM_VERSION}

# Allow normal users to read /root (as the generator is installed
# in /root/.cache, for example)
RUN chmod a+x /root

# The build-time pnpm keeps global packages and their binaries in
# /usr/local/pnpm. PNPM_HOME is only set while installing, so at run time pnpm
# keeps the versions it downloads under the writable HOME.
ENV PATH=$PATH:/usr/local/pnpm

# Install Node generation tools. pnpm 10 skips dependency build scripts; none
# of them are needed to generate code. The package store is only needed while
# installing, so it is removed afterwards.
RUN export HOME=/root PNPM_HOME=/usr/local/pnpm && \
    pnpm add -g --store-dir /tmp/pnpm-store \
        gapic-node-processing@${NODEJS_GAPIC_NODE_PROCESSING_VERSION} \
        gapic-tools@${NODEJS_GAPIC_TOOLS_VERSION} \
        protobufjs-cli@${NODEJS_PROTOBUFJS_CLI_VERSION} \
        protobufjs@${NODEJS_PROTOBUFJS_VERSION} && \
    curl -fsSL --retry 5 --retry-delay 15 -o /tmp/generator.tar.gz \
        https://github.com/googleapis/google-cloud-node/archive/${NODEJS_GAPIC_GENERATOR_VERSION}.tar.gz && \
    echo "${NODEJS_GAPIC_GENERATOR_CHECKSUM} /tmp/generator.tar.gz" | sha256sum -c - && \
    # The generator is installed in /opt (rather than a transient /tmp) and linked via
    # `pnpm add -g` to preserve its node_modules. This is necessary because google-gax is
    # defined as a devDependency in the generator but is required at startup. Deleting the
    # source directory would break the global installation with MODULE_NOT_FOUND.
    mkdir -p /opt/gapic-generator-typescript && \
    tar -xzf /tmp/generator.tar.gz -C /opt/gapic-generator-typescript --strip-components=4 google-cloud-node-${NODEJS_GAPIC_GENERATOR_VERSION}/core/generator/gapic-generator-typescript && \
    rm /tmp/generator.tar.gz && \
    cd /opt/gapic-generator-typescript && \
    pnpm install --store-dir /tmp/pnpm-store && \
    ./node_modules/.bin/tsc && \
    cp -a templates protos build/ && \
    pnpm add -g --store-dir /tmp/pnpm-store "$PWD" && \
    rm -rf /tmp/pnpm-store

# Fail the build if a runtime the image promises is missing.
RUN --mount=type=tmpfs,target=/home/librarian \
    node --version && pnpm --version

# ============== Java Stage ==============
FROM base AS java
ARG JAVA_VERSION
ARG JAVA_BUILD
ARG JAVA_CHECKSUM
ARG JAVA_MAVEN_VERSION
ARG JAVA_MAVEN_CHECKSUM

# Install the Temurin JDK and Maven from the official binaries.
RUN curl -fsSL --retry 5 --retry-delay 15 -o /tmp/jdk.tar.gz \
        https://api.adoptium.net/v3/binary/version/jdk-${JAVA_VERSION}%2B${JAVA_BUILD}/linux/x64/jdk/hotspot/normal/eclipse && \
    echo "${JAVA_CHECKSUM} /tmp/jdk.tar.gz" | sha256sum -c - && \
    mkdir -p /opt/java && \
    tar -xzf /tmp/jdk.tar.gz -C /opt/java --strip-components=1 && \
    rm /tmp/jdk.tar.gz
RUN curl -fsSL --retry 5 --retry-delay 15 -o /tmp/maven.tar.gz \
        https://archive.apache.org/dist/maven/maven-3/${JAVA_MAVEN_VERSION}/binaries/apache-maven-${JAVA_MAVEN_VERSION}-bin.tar.gz && \
    echo "${JAVA_MAVEN_CHECKSUM} /tmp/maven.tar.gz" | sha512sum -c - && \
    mkdir -p /opt/maven && \
    tar -xzf /tmp/maven.tar.gz -C /opt/maven --strip-components=1 && \
    rm /tmp/maven.tar.gz
ENV JAVA_HOME=/opt/java
ENV PATH=/opt/java/bin:$PATH

# The JVM reads its home directory from /etc/passwd, which has no entry for
# most UIDs that run the image. This mvn passes HOME instead, so Maven uses the
# local repository that Librarian reads.
RUN printf '#!/bin/sh\nexec /opt/maven/bin/mvn -Duser.home="$HOME" "$@"\n' > /usr/local/bin/mvn && \
    chmod 755 /usr/local/bin/mvn

# Setup dynamic target directories inside the persistent volume.
ENV LIBRARIAN_INSTALL_DIR=/usr/local/java_tools/bin
ENV PATH=${LIBRARIAN_INSTALL_DIR}:${PATH}

# Persist repository downloads inside the volume cache to save bandwidth and build time
ENV LIBRARIAN_CACHE=/usr/local/java_tools/cache

# Create java_tools directory and grant broad write permissions.
# This allows any arbitrary UID running the container (like librarianops) to
# execute `librarian install java` and write tools to the global path.
RUN mkdir -p /usr/local/java_tools && \
    chmod -R a+rwx /usr/local/java_tools

# Fail the build if a runtime the image promises is missing.
RUN --mount=type=tmpfs,target=/home/librarian \
    java -version && mvn -version
