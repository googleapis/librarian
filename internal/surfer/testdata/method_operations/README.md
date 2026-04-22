# Operations Commands

Operations commands.

## Proto Requirements:
* Long running methods must return long running operation

```
// Delete a single resource.
rpc DeleteResource(.method_operations.v1.DeleteResourceRequest)
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

## Service Config:
* Service config can specify operation as a mixin [AIP-4234](https://google.aip.dev/client-libraries/4234).
* Service config must also contain http rules for the operations methods.

```
apis:
- name: method_operations.v1.MethodOperations
- name: google.longrunning.Operations

http:
  fully_decode_reserved_expansion: true
  rules:
  - selector: google.longrunning.Operations.GetOperation
    get: '/v1/{name=shelves/*/operations/*}'
  - selector: google.longrunning.Operations.ListOperations
    get: '/v1/{name=shelves/*}/operations'
  - selector: google.longrunning.Operations.DeleteOperation
    delete: '/v1/{name=shelves/*/operations/*}'
  - selector: google.longrunning.Operations.CancelOperation
    post: '/v1/{name=shelves/*/operations/*}:cancel'
    body: '*'
```

## Gcloud config:
* Must specify generate_operations as True. (Defaults to True if not specified)

```
config_version: v2.0
service_name: methodoperations.googleapis.com
generate_operations: true
```

## YAML Output:
* Each of the operations method exposed in the service config, will generate
an operations method. For example, the above will generate cancel.yaml,
delete.yaml, describe.yaml, list.yaml, wait.yaml.

## Gcloud UX:
* User are able to cancel, delete, describe, list, and wait for operations
resources.

```
$ gcloud method-operations resources operations list ...
```
