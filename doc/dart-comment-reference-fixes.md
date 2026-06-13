# Dart Comment Reference Fixes Heuristics

This document summarizes the heuristics implemented in the Go-based Dart generator to sanitize documentation comments and resolve `comment_references` lint warnings.

## Heuristics Table

| Case | Original Example | Detection Strategy | Replacement / Action | Result |
| :--- | :--- | :--- | :--- | :--- |
| **Quoted URL with Brackets** | `"https://[service.name]/..."` | `"(https?://[^"]*\[[^"]*)"` | **Wrapped and Stripped.** Wraps in backticks and removes ref targets. | `` `https://[service.name]/...` `` |
| **Non-quoted URL with Brackets** | `https://[Service_name]/...` | `(^\|\s)(https?://[^\s\[]*\[[^\s]*)` | **Wrapped and Stripped.** Wraps in backticks and removes ref targets. | `` `https://[Service_name]/...` `` |
| **Valid Markdown Link** | `[audiences](https://...)` | `\[([\w\d\._]+)\]` followed by `(` | **Skipped.** Left as is to preserve valid external links. | `[audiences](https://...)` |
| **Already in Code Font** | `` `[Ref]` `` | `\[([\w\d\._]+)\]` preceded by `` ` `` | **Skipped.** Left as is to avoid double wrapping. | `` `[Ref]` `` |
| **Known Class Reference** | `[ValidRef]` | `\[([\w\d\._]+)\]` and `classExists("ValidRef")` is true | **Left as is.** Valid Dartdoc link to a class. | `[ValidRef]` |
| **Resolved Field Mapping** | `[Endpoint.dedicated_endpoint_dns]` | `\[([\w\d\._]+)\]` and field found in message | **CamelCased.** Maps snake_case field to Dart camelCase. | `[Endpoint.dedicatedEndpointDns]` |
| **Array Access** | `grounding_chunk[1]` | `([\w\d_]+)\[(\d+)\]` | **Wrapped.** Replaced with entire access in backticks. | `` `grounding_chunk[1]` `` |
| **Array Literal** | `[1,3,4]` | `\[\d+(,\d+)*\]` | **Wrapped.** Replaced with literal in backticks. | `` `[1,3,4]` `` |
| **Shortcut Reference Link** | `[Locations][]` | `\[([^\]]+)\]\[([\w\d\._]*)\]` and ref is empty | **Wrapped.** Replaced with just the label in backticks. | `` `Locations` `` |
| **Full Ref Link (Phrase)** | `[User-Managed Replica][Ref]` | `\[([^\]]+)\]\[([\w\d\._]*)\]` and label has spaces | **Stripped.** Keeps only the label as plain text. | `User-Managed Replica` |
| **Full Ref Link (Symbol)** | `[UpdateCmekSettings][Ref]` | `\[([^\]]+)\]\[([\w\d\._]*)\]` and label has no spaces | **Wrapped.** Replaced with label in backticks. | `` `UpdateCmekSettings` `` |
| **HTML Tags** | `<sometag>` | `<` and `>` | **Escaped.** Replaces with `&lt;` and `&gt;`. | `&lt;sometag&gt;` |
| **Fallback (Unresolved)** | `[UnknownRef]` | `\[([\w\d\._]+)\]` and no resolution found | **Wrapped.** Backticks applied while retaining brackets. | `` `[UnknownRef]` `` |
| **Double Backticks Cleanup** | ````Distribution```` | `\`\`([\\w\\d_]+)\`\`` | **Unwrapped.** Replaces double backticks with single backticks. | `` `Distribution` `` |
