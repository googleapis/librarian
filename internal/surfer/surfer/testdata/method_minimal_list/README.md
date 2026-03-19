# List Methods

List commands without `filter` and `order_by` fields.

## Proto Requirements [AIP-132](https://google.aip.dev/132):
* List methods must contain `page_size` and `page_token` fields
* List methods may contain `order_by` and `filter` fields

```
// The request structure for the ListResources method.
message ListResourcesRequest {
  // The parent of the resource.
  string parent = 1 [
    (.google.api.field_behavior) = REQUIRED,
    (.google.api.resource_reference) = {
      child_type: "example.googleapis.com/Resource"
    }
  ];

  // The maximum number of resources to send per page.
  int32 page_size = 2;

  // The page token: If the next_page_token from a previous response
  // is provided, this request will send the subsequent page.
  string page_token = 3;
}
```

## YAML Output:
* List command flags filter, limit, page-size, and sort-by flags are
automatically added to list commands and not included in yaml output.

## Gcloud UX:
* Gcloud automatically makes multiple list requests until all results are
returned or we reach the `limit` flag value.
* Regardless of whether the request has a `order_by` or `filter` fields,
`order_by` and `filter` are done on the client side by gcloud to keep the UX
consistent.

```
$ gcloud method-minimal-list resources list ...
```
