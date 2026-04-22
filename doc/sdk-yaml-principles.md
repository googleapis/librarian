# Principles for sdk.yaml

Librarian relies on three primary configuration files to manage service
behavior and code generation. These files operate in a layered approach, moving
from general service definitions to language-specific overrides.

- Service config: contains the core service configuration information, as
  defined by API teams
- `sdk.yaml`: language-neutral configuration, which overrides existing values
  in the service config or provides additional parameters
- `librarian.yaml`: language-specific configuration tailored to client
  libraries for a specific language

This document outlines the principles for managing the
[sdk.yaml](internal/serviceconfig/sdk.yaml) file. This file primarily contains
exceptions to the default behavior of Librarian.

## Purpose

Librarian relies on conventions and discovery logic to determine how to
generate and release clients for Google Cloud and other APIs. By default, these
are the principles that librarian follows:

- All APIs under `google/cloud` that are GA and contain a publishing section
  ([example](https://github.com/googleapis/googleapis/blob/cf027ac51c71290c7357e4c98cf4e6dbb0157346/google/cloud/secretmanager/v1/secretmanager_v1.yaml#L44))
  should have a client library for every language.
- Any API that falls outside of that range must be explicitly listed in
  `sdk.yaml`.

See the [sdk.yaml documentation](https://github.com/googleapis/librarian/blob/main/doc/api-allowlist-schema.md)
for additional information.

However, some APIs require deviations from these defaults. `sdk.yaml` serves as
the central repository for these intentional exceptions.

## Key Use Cases and Examples

### 1. Language-agnostic configuration

Configuration that apply to the API on a product level are generally
language-agnostic and should be incorporated into the `sdk.yaml`, and **not**
as a language-specific option in `librarian.yaml`. For example, the Python
library property `product_documentation_override` pertains to the API library
on a product level, and is not a Python library specific feature.

Conversely, when a language needs a setting that is unique to that language’s
library construction, it should be a configuration property in the
language-specific library settings. For example, the Python library property
`default_version` is unique to how Python libraries are packaged and consumed.

### 2. Restricting Languages for Non-Cloud APIs

By default, Librarian will allow client generation in all supported languages
for APIs under `google/cloud`. For APIs that live outside the `google/cloud`
path but still use Librarian's infrastructure, it is often desirable to
restrict client generation to a specific set of languages.

**Example:**
```yaml
- path: google/ai/generativelanguage/v1
  languages:
    - go
    - nodejs
    - python
```
This ensures that for the Gemini Generative Language API, only Go, Node.js,
and Python clients are generated, even if Librarian supports others.

### 3. Overriding Release Levels

Librarian derives the release level (preview, stable) of a client based on the
API version (e.g., `v1` is usually stable, `v1alpha` is preview). If an API
needs to release a client at a level that does not conform to this derivation
logic, it must be explicitly set.

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
Here, the Java client for this alpha API is explicitly kept in "preview" state
or similar override.

### 4. Overriding Transports

Librarian defaults to using gRPC & REST. If a specific transport must be used
for all languages or a subset of languages, it can be overridden.

**Example:**
```yaml
- path: google/ads/admanager/v1
  transports:
    all: rest
```
This forces the use of REST for all languages for the Ad Manager API.

### 5. Overriding Rest Numeric Enums

Some languages might need to skip REST numeric enums due to compatibility or
legacy reasons.

**Example:**
```yaml
- path: google/api/serviceusage/v1
  skip_rest_numeric_enums:
    - go
```

## Management

- This file is managed manually by the Librarian team.
- When we identify an API that requires an exception that cannot be inferred
  via standard discovery, an entry should be added or updated in this file.
- All changes to the schema should first be proposed on the
  [issue tracker](https://github.com/googleapis/librarian/issues) and reviewed
  by the Librarian team to ensure they align with these principles.
- Changes to `sdk.yaml` won't take effect during generation unless the
  Librarian version is bumped in the `librarian.yaml` file of the language
  repositories.
- We should strive to minimize the number of entries in `sdk.yaml` and reduce
  the number of exceptional entries over time.
