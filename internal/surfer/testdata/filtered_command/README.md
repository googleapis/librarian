# Filtering Commands

Filtering which commands a contributor wants to generate. Use the
`generation_filter` definition, which will exclude by default unless the
`include: true` attribute is added. The attributes for filtering are:

-   [`resource_generation_filters`](https://source.corp.google.com/piper///depot/google3/cloud/sdk/tools/gen_sfc/config/gcloud_config_schema_v2.yaml;l=255;rcl=663465290).
    Used to filter a resource that corresponds to a command group (`__init__.py`).
    <br/> *Each API definition will have one resource*.

-   [`method_generation_filters`](https://source.corp.google.com/piper///depot/google3/cloud/sdk/tools/gen_sfc/config/gcloud_config_schema_v2.yaml;l=261;rcl=663465290).
    Used to filter an RPC that corresponds to a command (`command.yaml`).
    <br/> *Each API definition can have multiple methods acting upon the resource*.

-   [`field_generation_filters`](https://source.corp.google.com/piper///depot/google3/cloud/sdk/tools/gen_sfc/config/gcloud_config_schema_v2.yaml;l=267;rcl=663465290).
    Used to filter a field of a resource that corresponds to a flag of a command.
    <br/> *Each API definition can have multiple fields for a resource*.

The filtering rules for generation in your service are:

*   Exclusion. Generate everything, but the excluded resources, methods, or
    fields defined.
*   Inclusion. Only generate the resources and methods defined and marked as
    included.
*   Exclusion or Inclusion. Only generate the resources and methods are marked
    as included and that are not defined as excluded.

## Proto Requirements:

*   Each command group corresponds to a resource.
*   Each command in group corresponds to method that is acting upon resource.

```
message Resource {
  option (.google.api.resource) = {
    type: "example.googleapis.com/Resource",
    pattern: [ "projects/{project}/locations/{location}/resources/{resource}" ]
  };

  // Resource name.
  string name = 1 [(.google.api.field_behavior) = IDENTIFIER];
}

service FilteredCommand {
  // Retrieve a collection of resources.
  rpc ListResources(.filtered_command.v1.ListResourcesRequest)
      returns (.filtered_command.v1.ListResourcesResponse) {
    option (.google.api.http) = {
      get: "/v1/{parent=projects/*/locations/*}/resources"
    };
    option (.google.api.method_resource) = {
      type: "example.googleapis.com/Resource"
    };
    option (.google.api.method_signature) = "parent";
  }

  // Retrieve a single resource.
  rpc GetResource(.filtered_command.v1.GetResourceRequest)
      returns (.filtered_command.v1.Resource) {
    option (.google.api.http) = {
      get: "/v1/{name=projects/*/locations/*/resources/*}"
    };
    option (.google.api.method_resource) = {
      type: "example.googleapis.com/Resource"
    };
    option (.google.api.method_signature) = "name";
  }

  // Retrieve a single resource.
  rpc GetFilteredResource(.filtered_command.v1.GetFilteredResourceRequest)
      returns (.filtered_command.v1.FilteredResource) {
    option (.google.api.http) = {
      get: "/v1/{name=projects/*/locations/*/filteredResources/*}"
    };
    option (.google.api.method_resource) = {
      type: "example.googleapis.com/FilteredResource"
    };
    option (.google.api.method_signature) = "name";
  }

  // Create a new resource.
  rpc CreateResource(.filtered_command.v1.CreateResourceRequest)
      returns (.filtered_command.v1.Resource) {
    option (.google.api.http) = {
      post: "/v1/{parent=projects/*/locations/*}/resources",
      body: "resource"
    };
    option (.google.api.method_resource) = {
      type: "example.googleapis.com/Resource"
    };
    option (.google.api.method_signature) = "parent,resource,resource_id";
  }

  // Update a single resource.
  rpc UpdateResource(.filtered_command.v1.UpdateResourceRequest)
      returns (.filtered_command.v1.Resource) {
    option (.google.api.http) = {
      patch: "/v1/{resource.name=projects/*/locations/*/resources/*}",
      body: "resource"
    };
    option (.google.api.method_resource) = {
      type: "example.googleapis.com/Resource"
    };
    option (.google.api.method_signature) = "resource,update_mask";
  }

  // Delete a single resource.
  rpc DeleteResource(.filtered_command.v1.DeleteResourceRequest)
      returns (.google.protobuf.Empty) {
    option (.google.api.http) = {
      delete: "/v1/{name=projects/*/locations/*/resources/*}"
    };
    option (.google.api.method_resource) = {
      type: "example.googleapis.com/Resource"
    };
    option (.google.api.method_signature) = "name";
  }
}
```

## gcloud config:

*   The filters shown below will ensure only the command associated with
    `filtered_command.v1.FilteredCommand.GetResource` will be generated.

```
method_generation_filters:
- selector: filtered_command.v1.FilteredCommand.ListResources
- selector: filtered_command.v1.FilteredCommand.DeleteResource
- selector: filtered_command.v1.FilteredCommand.CreateResource
- selector: filtered_command.v1.FilteredCommand.UpdateResource
resource_generation_filters:
- selector: filtered_command.v1.FilteredResource
```
