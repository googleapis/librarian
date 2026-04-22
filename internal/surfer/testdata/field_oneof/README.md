# Oneof groups

CRUDL commands for mutex groups.

## Proto Requirements [AIP-146](https://google.aip.dev/146):
* _oneof_ is used to annotate union type

```
message Resource {
  ...

  // Mutex group.
  oneof mutex {
    // Option #1.
    string one = 2;

    // Option #2.
    string two = 3;
  }
}
```

## YAML Output
* _oneof_ creates a mutex argument group

```
arguments:
  params:
  - group:
      mutex: true
      help_text: |-
        Arguments for the mutex.
      params:
      - arg_name: one
        api_field: resource.one
        required: false
        repeated: false
        help_text: |-
          Option #1.
      - arg_name: two
        api_field: resource.two
        required: false
        repeated: false
        help_text: |-
```

## Gcloud UX
* Users can only specify one item in an argument group or they will receive a
client side error.

```
$ gcloud field-oneof resources create --one=foo

$ gcloud field-oneof resources create --two=bar
```
