# Confirmation Prompt

CRUDL commands that need a confirmation prompt.

## Gcloud Config:
* Use a selector to determine which command needs a confirmation prompt.

```
input_dialogs:
- selector: confirmation_prompt.v1.ConfirmationPrompt.DeleteResource
  confirmation_prompt: Are you super duper sure you want to delete resource?
```

## YAML Output:
* Outputs `input.confirmation_prompt` in command.

```
input:
  confirmation_prompt: Are you super duper sure you want to delete resource?
```

## Gcloud UX:
Confirmation prompt will automatically print after user runs command.

```
$ gcloud confirmation-prompt resources delete ...

Are you super duper sure you want to delete resource?
```
