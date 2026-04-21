# Principles for internal/serviceconfig/sdk.yaml

This document outlines the principles for managing the `internal/serviceconfig/sdk.yaml` file. This file primarily contains exceptions to the default behavior of Librarian.

## Purpose

Librarian relies on conventions and discovery logic to determine how to generate and release clients for Google Cloud and other APIs. However, some APIs require deviations from these defaults. `sdk.yaml` serves as the central repository for these intentional exceptions.

## Key Use Cases and Examples

### 1. Restricting Languages for Non-Cloud APIs

By default, Librarian might attempt to generate clients for all supported languages. For APIs that live outside the `google/cloud` path but still use Librarian's infrastructure, it is often desirable to restrict client generation to a specific set of languages.

**Example:**
```yaml
- path: google/ai/generativelanguage/v1
  languages:
    - go
    - nodejs
    - python
```
This ensures that for the Gemini Generative Language API, only Go, Node.js, and Python clients are generated, even if Librarian supports others.

### 2. Overriding Release Levels

Librarian derives the release level (preview, stable) of a client based on the API version (e.g., `v1` is usually stable, `v1alpha` is preview). If an API needs to release a client at a level that does not conform to this derivation logic, it must be explicitly set.

**Example:**
```yaml
- path: google/analytics/admin/v1alpha
  languages:
    - go
    - java
    - nodejs
    - python
  release_level:
    java: preview
```
Here, the Java client for this alpha API is explicitly kept in "preview" state or similar override.

### 3. Overriding Transports

Librarian usually detects whether to use gRPC or REST based on service configuration. If a specific transport must be used for all languages or a subset of languages, it can be overridden.

**Example:**
```yaml
- path: google/ads/admanager/v1
  transports:
    all: rest
```
This forces the use of REST for all languages for the Ad Manager API.

### 4. Overriding Rest Numeric Enums

Some languages might need to skip REST numeric enums due to compatibility or legacy reasons.

**Example:**
```yaml
- path: google/api/serviceusage/v1
  skip_rest_numeric_enums:
    - go
```

## Management

- This file is managed manually.
- When we identify an API that requires an exception that cannot be inferred via standard discovery, an entry should be added or updated in this file.
- Changes should be reviewed by the Librarian team to ensure they align with these principles.
- Changes to `sdk.yaml` won't take affect during generation unless Librarian version is bumped in the `librarian.yaml` file of the language repositories.
- We should strive to minimize the number of entries in `sdk.yaml` and reduce the number of exceptional entries over time.
