# Primitive Field Types

CRUDL commands with primitive fields.

## Proto Requirements [AIP-203](https://google.aip.dev/203):
* Fields are prefixed with a scalar type ie string, uint32, int32, bytes,
float, and bool

```
message Resource {
  ...

  // String field.
  string string_field = 2;

  // Uint field.
  uint32 uint_field = 3;

  // Int field.
  int32 int_field = 4;

  // Bytes field.
  bytes bytes_field = 5;

  // Float field.
  float float_field = 6;

  // Bool field.
  bool bool_field = 7;
}
```

## YAML Output:
* Each field gets their own gcloud flag with the correct api_field.
* Type is included, however, type can be deduced from api_field.
* NOTE: _byte_ type is currently not handled and is skipped in output.

```
arguments:
  params:
  ...
  - arg_name: string-field
    api_field: resource.stringField
    required: false
    repeated: false
    help_text: |-
      String field.
  - arg_name: uint-field
    api_field: resource.uintField
    required: false
    repeated: false
    type: int
    help_text: |-
      Uint field.
  - arg_name: int-field
    api_field: resource.intField
    required: false
    repeated: false
    type: int
    help_text: |-
      Int field.
  - arg_name: float-field
    api_field: resource.floatField
    required: false
    repeated: false
    type: float
    help_text: |-
      Float field.
  - arg_name: bool-field
    api_field: resource.boolField
    action: store_true
    required: false
    type: bool
    help_text: |-
      Bool field.
```

## Gcloud UX:
* User can provide string and the type will be automatically transformed.
* Boolean flags are store_true flags.

```
$ gcloud field-simple-types resources create \
    --string-field=foo --uint-field=1 --int-field=2 --bool-field
```
