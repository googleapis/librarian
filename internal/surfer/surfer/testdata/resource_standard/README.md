# Standard Resources

CRUDL commands for single pattern resources.

## Proto Requirements [AIP-122](https://google.aip.dev/122), [AIP-123](https://google.aip.dev/123):
* Gcloud: APIs must specify pattern associated with resource type using the
google.api.resource annotation
* Resource patterns must follow plural/{singular} pattern
* Resources can be shortened to redundant form i.e.
_users/vhugo1802/userEvents/birthday-dinner-226_ can be shortened to
_users/vhugo1802/events/birthday-dinner-226_. However, the underlying message
is still UserEvent resource

```
message Resource {
  option (.google.api.resource) = {
    type: "example.googleapis.com/Resource",
    pattern: ["projects/{project}/locations/{location}/resources/{resource}"]
  };

  // Resource name.
  string name = 1 [(.google.api.field_behavior) = IDENTIFIER];
}
```

**NOTE:** Singleton resources are not supported in autogen at this time.

## YAML Output:
* The _name_ field triggers the resource argument to be rendered as a CLI
flag. The pattern specified in the message's resource descriptor, is used to
determine the collection and attributes of the resource.
* If project, organization, folder, location, region, and zone are not included
in the command hierarchy i.e. the command is `gcloud surface resource`
not `gcloud surface project resource`.

```
arguments:
  params:
  - help_text: |-
      The resource name of the resource within a service.
    is_positional: true
    resource_spec: !REF googlecloudsdk.command_lib.standard_resource.generated_resources:project_location_resource
    required: true
```

## Gcloud UX:
The resource arguments can either be specified as the full uri or as individual
flags.

```
$ gcloud library books describe shelves/my-shelf/books/my-book
$ gcloud library books describe my-book --shelf=my-shelf
```
