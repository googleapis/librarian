# Resource Reference

CRUDL commands with message fields that reference another resource.

## Proto Requirements [AIP-124](https://google.aip.dev/124), [AIP-4231](https://google.aip.dev/client-libraries/4231):
* Resources map reference other resources using resource_reference annotation
* Gcloud: the type's pattern must be specified somewhere through a
resource_definition or gcloud config

```
option (.google.api.resource_definition) = {
  type: "example.googleapis.com/Reference",
  pattern: [ "projects/{project}/references/{reference}" ]
};

message Resource {
  option (.google.api.resource) = {
    type: "example.googleapis.com/Resource",
    pattern: [ "projects/{project}/locations/{location}/resources/{resource}" ]
  };

  // Resource name.
  string name = 1 [(.google.api.field_behavior) = IDENTIFIER];

  // Reference to another resource
  string reference = 2 [(.google.api.resource_reference) = {
    type: "example.googleapis.com/Reference"
  }];

  // Reference to another resource
  // (Pattern defined in gcloud config)
  string another_reference = 3 [(.google.api.resource_reference) = {
    type: "example.googleapis.com/AnotherReference"
  }];

  ...
}
```

**NOTE:** Singleton resources are not supported in autogen at this time.
Resources where the parent is a singleton are supported i.e.
`projects/{project}/global/networks/{network}`.

## YAML Output:
* Renders string fields with resource reference as a resource argument
* Resource_method_params is used to map the resource argument to the correct
location

```
arguments:
  params:
  - ...
  - arg_name: reference
    help_text: |-
      Reference to another resource
    is_positional: false
    resource_spec: !REF googlecloudsdk.command_lib.resource_reference.v1_resources:project_reference
    resource_method_params:
      resource.reference: '{__relative_name__}'
    required: false
  - arg_name: another-reference
    help_text: |-
      Reference to another resource
    is_positional: false
    resource_spec: !REF googlecloudsdk.command_lib.resource_reference.v1_resources:project_location_another
    resource_method_params:
      resource.anotherReference: '{__relative_name__}'
```

## Gcloud UX:
* Users are able to able define the resource reference using flags or fully
specified uri

```
$ gcloud resource-reference resources create my-resource \
  --reference=my-reference --project=my-project

$ gcloud resource-reference resources create my-resource \
  --reference=projects/my-project/references/my-reference
```
