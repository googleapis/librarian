# Repository/Library Onboarding Guide

This guide should be followed when onboarding new repositories/libraries.

## Repository Setup:
1) Add .librarian directory to your repository with appropriate configuration files. See details [here](https://github.com/googleapis/librarian/blob/main/doc/language-onboarding.md#configuration-files)
2) You should only start with 1 library to validate the flow (follow instructions below)
3) If your repository contains multiple libraries, start ramping up slowly until all libraries are in your state.yaml file and have migrated to librarian.
4) To complete onboarding you should run the librarian test-container generate (WIP) command to validate that all libraries are getting generated correctly.

## Library Setup:
1) Ensure all OwlBot PRs for that library have been merged and then release the library using a release-please PR
2) Remove the library from your OwlBot config
    - For a monolithic config remove all mentions of the library from your .Owlbot.yaml config file
    - For a single library repository remove the .Owlbot.yaml config file itself 
3) Remove the library from your release-please config
    - For a monolithic repo remove the path entry for the library in your release-please-config.json and .release-please-manifest.json files
    - For a single library repository, remove all the release-please config (.github/release-please.yml, release-please-config.json if it exists, .release-please-manifest.json if it exists)
4) There is no requirement to STOP using library specific owlbot post processing files as part of this migration.  However while migrating please open an issue in your generator repository for any improvements that could reduce your library post processing logic.  
5) Add your library to your [libraries object](https://github.com/googleapis/librarian/blob/main/doc/state-schema.md#libraries-object) in your [state.yaml](https://github.com/googleapis/librarian/blob/main/doc/state-schema.md#stateyaml-schema) file
6) Run [librarian generate command](https://github.com/googleapis/librarian/blob/main/doc/cli-commands.md#generate-command).  The output should be 0 diff, check with your language lead/generator owner if this is not the case.
7) Be aware of the `generate_blocked` and `release_blocked` fields. If these are set to `true` and automation is enabled for the repository ([check here](https://github.com/googleapis/librarian/blob/main/internal/automation/prod/repositories.yaml)), then generate and release PRs will be created and merged automatically. If these actions are blocked, or your repository is not set up for automation, you will have to perform these actions manually. See this [guide](https://github.com/googleapis/librarian/blob/main/doc/library-maintainer-guide.md) for details
