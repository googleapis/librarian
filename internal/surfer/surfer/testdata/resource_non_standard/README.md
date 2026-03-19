# Non-standard Resources

CRUDL command for resources with parents other than project, organization,
folder, or location.

## Proto Requirements [AIP-122](https://google.aip.dev/122), [AIP-123](https://google.aip.dev/123):
* Gcloud: APIs must specify pattern associated with resource type using the
google.api.resource annotation
* Resource patterns must follow plural/{singular} pattern

```
message Resource {
  option (.google.api.resource) = {
    type: "example.googleapis.com/Resource",
    pattern: [
      "projects/{project}/locations/{location}/apis/{api}/resources/{resource}"
    ]
  };

  // Resource name.
  string name = 1 [(.google.api.field_behavior) = IDENTIFIER];
}
```

## YAML Output:
* The _name_ field triggers the resource argument to be rendered as a CLI
flag. The pattern specified in the message's resource descriptor, is used to
determine the collection and attributes of the resource.

```
arguments:
  params:
  - help_text: |-
      The resource name of the resource within a service.
    is_positional: true
    resource_spec: !REF googlecloudsdk.command_lib.non_standard_resource.generated_resources:project_location_resource
    required: true
```

## Gcloud UX:
* Each resource in the hierarchy that is not standard, is given a command group
* The resource arguments can either be specified as the full uri or as
individual flags.

```
$ gcloud resource-non-standard apis describe my-api

$ gcloud resource-non-standard apis resources describe my-resource \
  --api=my-api
```
