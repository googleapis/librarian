# Regional Endpoints

Used to run workloads in a manner that complies with data residency and data
sovereignty requirements, where your request traffic is routed directly to
the region specified in the endpoint.

## gcloud.yaml:
Autogen will automatically determine whether the service supports regional
endpoints based off of the `endpoints` section of the service config.
However, contributors can override the support by specifying whether
regional endpoints are `REQUIRED`, `SUPPORTED`, or `NOT_SUPPORTED`.

* `REQUIRED`: If a regional endpoint is not accurately specified, then the
user will receive an error.
* `SUPPORTED`: If a resional endpoint is not accurately specified, then the
user will default to using a global endpoint.
* `NOT_SUPPORTED`: User is only able to use global endpoints.

```
config_version: v2.0
service_name: regionalendpoints.googleapis.com
regional_endpoint_compatibility: REQUIRED
```

`add_regional_endpoint_flag` determines which commands should add an
`--endpoint-location` flag. Most commands do need the `--endpoint-location`
flag.
```
apis:
- name: RegionalEndpoints
  regional_endpoint_config:
  - selector: regional_endpoints.v1.Resource
    add_regional_endpoint_flag: true  # optional
```

## YAML Output:
`regional_endpoint_compatibility` field is updated the correct compatibility.
`NOT_SUPPORTED` defaults to regional_ednpoint_compatibility to not being
specified at all (default).

```
request:
  api_version: v1
  collection:
  - regionalendpoints.projects.locations.resources
  regional_endpoint_compatibility: REQUIRED | SUPPORTED | null
```

## Gcloud UX:
If a user set `--endpoint-mode` to `regional` or `regional-preferred`, a user
is able to specify the endpoint using `location` | `region` | `zone`
attribute of the primary resource the operation is being performed on.
`--endpoint-location` flag is used if the command does not have a resource
or if the resource does not contain a `location` | `region` | `zone` attribute.

In both of the examples below, `my-location` is used to determine the regional
endpoint to make the API request to i.e.
https://SERVICE_NAME.my-location.rep.googleapis.com/.

```
$ gcloud regional-endpoins-regional-supported projects/my-project/locations/my-location/resources/my-resource

$ gcloud regional-endpoins-regional-supported projects/my-project/resources/my-resource --endpoint-location=my-location
```

**NOTE:** `--endpoint-location` is only added to commands that set
`add_regional_endpoint_flag` to true. Most commands do not need the
`--endpoint-location` flag.


