# Private Preview

If you want to be able to test a feature behind a visibility label, gen_sfc
can generate the command or flag for you and set it to hidden.

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

## Build configuration:

To expose features behind a visility label, you must update the apitools
discovery doc to include the required visibility.

```
api_discovery(
    # target_name = "privatepreview_TRUSTED_TESTER_google_rest_v1"
    name = "privatepreview",
    schema = "GOOGLE_REST",
    version = "v1",
    visibility_labels = ["TRUSTED_TESTER"],
)
```

## YAML Output:
If a service or method is behind a visibility label, the corresponding command
will be generated but hidden. If a message, field, or enum is behind a
visibility label, then that flag will be generated but hidden.

Hidden means anyone can use the feature but they will not show up in public
docs.

```
- arg_name: public-field
  api_field: publicField
  required: false
  repeated: false
  help_text: |-
    Always visible field.
- arg_name: private-field
  api_field: privateField
  required: false
  repeated: false
  hidden: true
  help_text: |-
    Private field.
```
