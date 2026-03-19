# Standard Resources

CRUDL commands for single pattern resources.

## Proto Requirements [AIP-202](https://google.aip.dev/202), [AIP-203](https://google.aip.dev/203), [AIP-213](https://google.aip.dev/213):

* Google.api.field_behavior must be specified and output REQUIRED, OPTIONAL,
  or OUTPUT_ONLY at a minimum
* IMMUTABLE can also be included
* Apply visibility labels for components that are meant to be internal.

```
message Resource {
  option (.google.api.resource) = {
    type: "example.googleapis.com/Resource",
    pattern: [ "projects/{project}/locations/{location}/resources/{resource}" ]
  };

  // Resource name.
  string name = 1 [(.google.api.field_behavior) = IDENTIFIER];

  // Output only field.
  string output_only_field = 2 [(.google.api.field_behavior) = OUTPUT_ONLY];

  // Visible private field.
  string internal_field = 3
      [(google.api.field_visibility).restriction = "GOOGLE_INTERNAL"];

  // Invisible private field
  string hidden_field = 4 [(google.api.field_visibility).restriction = "OTHER"];

  // Required field.
  string required_field = 5 [(google.api.field_behavior) = REQUIRED];

  // Immutable field.
  string immutable_field = 6 [(google.api.field_behavior) = IMMUTABLE];
}
```

## Build rule:

* Visibility labels must be added to build rule if we wanted it included in
  in output.

```
# buildifier: disable=duplicated-name
api_discovery(
    # target_name = "fieldattributes_GOOGLE_INTERNAL_google_rest_v1"
    name = "fieldattributes",
    schema = "GOOGLE_REST",
    version = "v1",
    visibility_labels = ["GOOGLE_INTERNAL"],
)

gcloud_api_client_source(
    name = "fieldattributes/v1",
    discovery = ":fieldattributes_GOOGLE_INTERNAL_google_rest_v1.json",
)
```

## YAML Output:

* Each field gets their own gcloud flag with the correct api_field.
* Flags default to optional but are specified as required for create commands
  if required in the proto.
* Only fields with visibility labels that match the build rule are generated.
* The flags generated from fields behind visibility labels are marked hidden
  in the CLI.
* OUTPUT_ONLY fields are not included as flags ever.
* IMMUTABLE fields are not included for update commands.

```
# create
arguments:
  params:
  ...
  - arg_name: internal-field
    api_field: resource.internalField
    required: false
    repeated: false
    help_text: |-
      Visible private field.
  - arg_name: required-field
    api_field: resource.requiredField
    required: true
    repeated: false
    help_text: |-
      Required field.
  - arg_name: immutable-field
    api_field: resource.immutableField
    required: false
    repeated: false
    help_text: |-
      Immutable field.

# Update
arguments:
  params:
  ...
  - arg_name: internal-field
    api_field: resource.internalField
    required: false
    repeated: false
    help_text: |-
      Visible private field.
  - arg_name: required-field
    api_field: resource.requiredField
    required: false
    repeated: false
    help_text: |-
      Required field.
```

## Gcloud UX:
User must specify required fields for create and can optionally provide them
for update.

```
$ gcloud field-attributes resources create --internal-field=string1 \
    --required-field=string2 --immutable-field=string3

$ gcloud field-attributes resources update --internal-field=string1 \
    --required-field=string2
```
