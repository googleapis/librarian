# Custom Methods

Resource or collection-based custom methods

## Proto Requirements [AIP-136](https://google.aip.dev/136):
* Custom methods must contain a HTTP URI with _:verb_.

```
// Retrieve a collection of resources.
rpc SearchResources(SearchResourcesRequest)
    returns (SearchResourcesResponse) {
  option (.google.api.http) = {
    get: "/v1/{parent=projects/*/locations/*}/resources:search"
  };
  option (google.api.method_signature) = "parent,target";
}

// Archives the given resource.
rpc ArchiveResource(ArchiveResourceRequest) returns (Resource) {
  option (google.api.http) = {
    post: "/v1/{name=projects/*/locations/*/resources/*}:archive"
    body: "*"
  };
}
```

## YAML Output go/gcloud:autogen-custom-command-generator-dd:
* The verb in the http uri becomes the name of the command.
* Flags are generated from request message fields as normal.
* Commands associated with a resource type are still placed in the same command
group.

```
arguments:
  params:
  - help_text: |-
      The parent of the resource.
    is_positional: false
    is_parent_resource: true
    is_primary_resource: true
    resource_spec: !REF googlecloudsdk.command_lib.method_custom.v1_resources:project_location
    required: true
  - arg_name: target
    api_field: target
    required: true
    repeated: false
    help_text: |-
      Target resource.
```

## Gcloud UX:
* Custom commands look similar to other resource based CRUDL commands where
a verb is being acted upon a resource.
* Commands with a pageToken request field will automatically make follow up
api requests until all results are returned or we reach the limit flag.
List-like flags (filer, limit, page-size, and sort-by) are automatically added
onto the command in gcloud.

```
$ gcloud method-custom resources search --target=foo
```
