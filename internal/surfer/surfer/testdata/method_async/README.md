# Aync Methods

CRUDL commands for long running operations.

## Proto Requirements [AIP-135](https://google.aip.dev/135):
* Create, update, and delete methods can be long running. Response to long
running methods must be `google.longrunning.Operation`.
* Operation info must contain `response_type` and `metadata_type`.

```
// Delete a single resource.
rpc DeleteResource(.method_async.v1.DeleteResourceRequest)
    returns (.google.longrunning.Operation) {
  option (.google.api.http) = {
    delete: "/v1/{name=projects/*/locations/*/resources/*}"
  };
  option (.google.api.method_resource) = {
    type: "example.googleapis.com/Resource"
  };
  option (.google.api.method_signature) = "name";
  option (.google.longrunning.operation_info) = {
    response_type: "google.protobuf.Empty",
    metadata_type: "OperationMetadata"
  };
}
```

## YAML Output:
* Generated commands looks similar to synchronous create, update, and delete
commands. Async section is added with the operations collections.

```
request:
  api_version: v1
  collection:
  - methodasync.projects.locations.resources
async:
  collection:
  - methodasync.projects.locations.operations
```

## Gcloud UX:
Gcloud polls until the operation is complete and returns the response.

```
$ gcloud method-async resources delete ...
```

The user can provide the `--async` flag to immediately return the operation.

```
$ gcloud method-async resources delete --async ...
```
