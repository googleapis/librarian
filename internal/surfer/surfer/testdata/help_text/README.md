# Help Text

gcloud commands must have help text.

## Proto Requirements:

*   The help text for Autogen gcloud commands, command groups, and flags will
    use proto comments above methods, messages, and fields by default.

```
// Resource message.
message Resource {
  option (.google.api.resource) = {
    type: "example.googleapis.com/Resource",
    pattern: [ "projects/{project}/locations/{location}/resources/{resource}" ]
  };

  // Resource name.
  string name = 1 [(.google.api.field_behavior) = IDENTIFIER];

  // A standard field for the resource (for help text).
  string display_name = 2;
}
```

## gcloud config

Help text can be overridden and prove examples and additional information in the
gcloud_config file using
`help_text_rule`
definition and the following attributes:

-   `service_rules`.
    Help text for the main surface (root `__init__.py`).
    <br/> *When multiple API interfaces are defined, it will use the first one.*.

-   `message_rules`.
    Help text for a resource that corresponds to a command group (`__init__.py`).
    <br/> *Each API definition will have one resource*.

-   `method_rules`.
    Help text for an RPC that corresponds to a command (`command.yaml`).
    <br/> *Each API definition can have multiple methods acting upon the resource*.

-   `field_rules`.
    Help text for a field of a resource that corresponds to a flag of a command.
    <br/> *Each API definition can have multiple fields for a resource*.

```
help_text:
  service_rules:
  - selector: helptext.v1.HelpText
    help_text:
      brief: Manage my help text
      description: |-
        Manage my help text resources
  method_rules:
  - selector: helptext.v1.HelpText.CreateResource
    help_text:
      brief: Create help text resource
      description: Create help text resource
      examples:
      - |-
        To create a help text resource:

        $ {command} --foo=bar
```

## YAML Output:
* Help text added to flags and command level.

```
- release_tracks:
  - GA
  auto_generated: true
  help_text:
    brief: Create help text resource!!
    description: Create help text resource!!
    examples: |-
      To create a help text resource:

      $ {command} --foo=bar
  arguments:
    params:
    ...
    - arg_name: display-name
      api_field: resource.displayName
      required: false
      repeated: false
      help_text: |-
        A standard field for the resource (for help text).
```

## gcloud UX:
* Can see output of help using `--help`

```
$ gcloud help-text resources create --help
```
