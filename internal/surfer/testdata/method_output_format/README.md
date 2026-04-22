# Output Format

Commands with default response format.

## Gcloud config:
* Contributors can specify default output format for commands. Most commonly,
list commands are given defaulted to a table format.

```
- name: MethodOutputFormat
  api_version: v1
  release_tracks:
  - GA
  output_formatting:
  - selector: method_output_format.v1.MethodOutputFormat.ListResources
    format: |-
      table(name,
            createTime,
            updateTime)
```

## YAML Output:
* Format in gcloud config is automatically added to relevant command.

```
output:
  format: |-
    table(name,
          createTime,
          updateTime)
```

## Gcloud UX:
* Command output will default to what is listed in the `output_format`.

```
$ gcloud method-output-format resources list ...
```
