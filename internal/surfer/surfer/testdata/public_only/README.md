# Public Only

Majority of the time, API teams do not want to expose any features behind
visibility labels to the public. By default, gen_sfc will not autogenerate
commands behind a visibility label.

## Proto Requirements:
APIs can use protos or service configs to add visibility labels to a service,
method, message, field, enum, or enum value
[AIP-185](https://google.aip.dev/185#visibility-based-versioning).

```
// Public message where the field is private in proto annotation.
message PublicProtoMessage {

  // Always visible field.
  string public_field = 1;

  // Private field.
  string private_field = 2
      [(google.api.field_visibility).restriction = "TRUSTED_TESTER"];

}
```

## YAML Output:
If a service or method is behind a visibility label, the corresponding command
will not be generated. If a message, field, or enum is behind a visibility
label, then that flag will not be generated.

```
- arg_name: public-field
  api_field: publicField
  required: false
  repeated: false
  help_text: |-
    Always visible field.
```
