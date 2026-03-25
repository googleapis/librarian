# Multitype Resources

CRUDL commands for multitype resources.

## Proto Requirements [AIP-127](https://google.aip.dev/127), [AIP-4231](https://google.aip.dev/client-libraries/4231):

#### Resource Message

* Occasionally, a resource may have more than one pattern. Common when a
resource can live under more than one parent

```
message Resource {
  option (.google.api.resource) = {
    type: "example.googleapis.com/Resource",
    pattern: [
      "projects/{project}/locations/{location}/resources/{resource}",
      "organizations/{organization}/locations/{location}/resources/{resource}",
      "folders/{folder}/locations/{location}/resources/{resource}"
    ]
  };

  // Resource name.
  string name = 1 [(.google.api.field_behavior) = IDENTIFIER];
}
```

#### RPC Methods

* RPC may define a number of additional bindings
* Gcloud: the patterns from additional_bindings must match the patterns of the
resource that is being operated on in the method.

```
// Create a new resource.
rpc CreateResource(.resource_multitype.v1.CreateResourceRequest)
    returns (.resource_multitype.v1.Resource) {
  option (.google.api.http) = {
    post: "/v1/{parent=projects/*/locations/*}/resources",
    body: "resource",
    additional_bindings: [
      {
        post: "/v1/{parent=organizations/*/locations/*}/resources",
        body: "resource"
      },
      {
        post: "/v1/{parent=folders/*/locations/*}/resources",
        body: "resource"
      }
    ]
  };
}
```

* RPC methods may also define a multiple RPCs with the same name but namespaced
under different services.

```
service ProjectService { ...

// Create a new resource in Project service.
rpc CreateResource(.resource_multitype.v1.CreateResourceRequest)
    returns (.resource_multitype.v1.Resource) {
  option (.google.api.http) = {
    post: "/v1/{parent=projects/*/locations/*}/resources",
    body: "resource",
  };
}

service OrganizationService { ...

// Create a new resource in Organization service.
rpc CreateResource(.resource_multitype.v1.CreateResourceRequest)
    returns (.resource_multitype.v1.Resource) {
  option (.google.api.http) = {
    post: "/v1/{parent=organizations/*/locations/*}/resources",
    body: "resource",
    additional_bindings: [
      {
        post: "/v1/{parent=organizations/*/locations/*}/resources",
        body: "resource"
      },
      {
        post: "/v1/{parent=folders/*/locations/*}/resources",
        body: "resource"
      }
    ]
  };
}
```

## YAML Output:

* Multiple patterns are added to the resources.yaml file.
* Auto completers are disabled for each multitype resource.

```
folder_or_organization_or_project_location_resource:
  name: resource
  plural_name: resources
  resources:
  - name: resource
    plural_name: resources
    collection: resourcemultitype.folders.locations.resources
    attributes:
    - *folder
    - *location
    - &resource
      parameter_name: resourcesId
      attribute_name: resource
      help: resources TBD
    disable_auto_completers: true
  - name: resource
    plural_name: resources
    collection: resourcemultitype.organizations.locations.resources
    attributes:
    - *organization
    - *location
    - *resource
    disable_auto_completers: true
  - name: resource
    plural_name: resources
    collection: resourcemultitype.projects.locations.resources
    attributes:
    - *project
    - *location
    - *resource
    disable_auto_completers: true
```

* Multitype commands are generated as single pattern commands.
`request.collection` includes each of the possible resource patterns.

```
- ...
  arguments:
    params:
    - help_text: |-
        Identifier. Resource name.
      is_positional: true
      request_id_field: resourceId
      resource_spec: !REF googlecloudsdk.command_lib.resource_multitype.v1_resources:folder_or_organization_or_project_location_resource
      required: true
  request:
    api_version: v1
    collection:
    - resourcemultitype.folders.locations.resources
    - resourcemultitype.organizations.locations.resources
    - resourcemultitype.projects.locations.resources
```

## Gcloud UX:

* Users can run commands on the parent and the child resource.

```
$ gcloud resource-multitype resources create my-resource \
  --location=my-location --project=my-project

$ gcloud resource-multitype resources create my-resource \
  --location=my-location --organization=my-organization

$ gcloud resource-multitype resources create my-resource
```
