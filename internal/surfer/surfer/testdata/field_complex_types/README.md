# Complex Field Types

CRUDL commands for complex field types.

## Proto Requirements [AIP-144](https://google.aip.dev/144), [AIP-146](https://google.aip.dev/146):
* _Repeated_ prefix is used to annotate a list
* Map fields are used whenever the keys have the same type but the key values
are unknown.

```
message Resource {
  option (.google.api.resource) = {
    type: "example.googleapis.com/Resource",
    pattern: [ "projects/{project}/locations/{location}/resources/{resource}" ]
  };

  // Resource name.
  string name = 1 [(.google.api.field_behavior) = IDENTIFIER];

  // Message field.
  Message my_message = 2;

  // Repeated field.
  repeated string repeated_field = 3;

  // Repeated message.
  repeated Message repeated_message = 4;

  // Map field.
  map<string, int32> map_field = 5;
}
```

## Gcloud config:
Message fields can also be annotated as `settable` and/or `clearable`. This
allows the contributor to specify an additional arg_object flag for
settable and additional clear flags for clearable.

```
generated_flags:
- selector: field_complex_types.v1.Resource.my_message
  settable: true
- selector: field_complex_types.v1.Resource.my_message
  clearable: true
```

If there is a string or bytes field that should be derived from the contents
of a file, a contributor can annotate the field as file type in the gcloud
config.

```
generated_flags:
- selector: field_complex_types.v1.Resource.repeated_file_field
  from_file: true
- selector: field_complex_types.v1.Message.file_field
  from_file: true
```

## YAML Output:
* Message fields traverse the message tree and prefix the flag name with the
path traveled.
* Repeated primitives are type ArgList.
* Repeated message fields and map fields are automatically ArgObject type
go/gcloud-arg-object.
* Update commands with ArgObject and ArgList flags automatically generate
update flags using _clearable_ go/gcloud-update-flags.

```
# update
arguments:
  params:
  ...
  - group:
      required: false
      help_text: |-
        Inner message.
      params:
      - arg_name: my-message-string-field
        api_field: resource.myMessage.stringField
        required: false
        repeated: false
        help_text: |-
          Message string field.
      - arg_name: my-message-int-field
        api_field: resource.myMessage.intField
        required: false
        repeated: false
        type: int
        help_text: |-
          Message int field.
  - arg_name: repeated-field
    api_field: resource.repeatedField
    required: false
    repeated: true
    help_text: |-
      Repeated field.
    clearable: true
  - arg_name: repeated-message
    api_field: resource.repeatedMessage
    required: false
    repeated: true
    help_text: |-
      Repeated message.
    clearable: true
    spec:
    - api_field: stringField
      help_text: |-
        Message string field.
    - api_field: intField
      help_text: |-
        Message int field.
  - arg_name: map-field
    api_field: resource.mapField
    required: false
    repeated: true
    help_text: |-
      Map field.
    clearable: true
    spec:
    - api_field: key
    - api_field: value
```

* Message fields that are annotated as `settable` will generate the necessary
arg group information to generate an arg_object flag. Empty message fields
will automatically generate a settable flag unless specified otherwise.
This ensures that the user can specify the empty message field now and be
backwards compatible if the message definition changes and adds fields.

```
- group:
    api_field: resource.myMessage
    arg_name: my-message
    settable: true
    required: false
    help_text: |-
      Inner message.
```

* Message fields that are annotated as `clearable` will generate the necessary
arg group information to generate a `--clear-{arg}` flag for update commands.
By default, root level message fields are also annotated as clearable. This
ensures that users can clear the field when updating a resource.

```
- group:
    api_field: resource.myMessage
    arg_name: my-message
    settable: true
    clearable: true
    required: false
    help_text: |-
      Inner message.
```

* Flags annotated as file flags, will be annotated as `file_type` and have
the suffix `-from-file`. File types nested inside an ArgObject will have the
suffix `FromFile`.

```
- arg_name: my-message-file-field-from-file
  api_field: resource.myMessage.fileField
  ...
  type: file_type
  help_text: |-
    Message file field.
```

## Gcloud UX:
* ArgObject types can be provided in shorthand, inline json, or file

```
$ gcloud field-comples-types resources create \
    --repeated-message=stringField=foo,intField=1 \
    --repeated-message=stringField=bar,intField=2

$ gcloud field-comples-types resources create \
    --repeated-message='[{"stringField": "foo", "intField": 1}, {"stringField": "bar", "intField": 2}]'
```

* Clearable update flags automatically add clear, update, add, and remove flags.

```
$ gcloud field-comples-types resources update --clear-resources
```

* Settable arg_group flags automatically generate an arg_object flag. It allows
users to specify the message in one arg_object flag or separate flags.

```
$ gcloud field-comples-types resources create \
    --my-message=stringField=foo,intField=1

$ gcloud field-comples-types resources create \
    --my-message-string-field=foo --my-message-int-field=1
```

* Clearable arg_group flags automatically generate a `--clear-{arg}` flag for
update commands.

```
$ gcloud field-comples-types resources update --clear-my-message
```

* File flags will contain `-from-file` or `FromFile` suffix
go/autogen-file-type.

```
$ gcloud field-comples-types resources create \
    --my-message-file-field-from-file=path_to_file
```

