# Librarian CLI

[![Go Reference](https://pkg.go.dev/badge/github.com/googleapis/librarian/cmd/librarian.svg)](https://pkg.go.dev/github.com/googleapis/librarian/cmd/librarian)
[![codecov](https://codecov.io/github/googleapis/librarian/graph/badge.svg?token=33d3L7Y0gN)](https://codecov.io/github/googleapis/librarian)

This repository contains code for a unified command line tool for
Google Cloud SDK client library configuration, generation and releasing.

See [CONTRIBUTING.md](CONTRIBUTING.md) for a guide to contributing to this repository,
and [the doc/ folder](doc/) for more detailed project documentation.

The Librarian project supports the Google Cloud SDK ecosystem, and
we do not *expect* it to be of use to external users. That is not
intended to discourage anyone from reading the code and documentation;
it's only to set expectations. (For example, we're unlikely to accept
feature requests for external use cases.)

## Installation

Install [Go](https://go.dev/doc/install) and make sure you have you have the
following in your `PATH`:

```sh
export PATH="$(go env GOPATH)/bin:$PATH"
```

Next install the CLI

```sh
go install github.com/googleapis/librarian/cmd/librarian@latest
```

## Running Librarian

To see the current set of commands available, run:

```sh
librarian -help
```

Use the `-help` or `-h` flag for any individual command to see detailed
documentation for its purpose and associated flags. For example:

```sh
librarian generate -h
```

Alternatively, if you prefer not to have librarian installed you can use the Go
command to run the latest released version:

```sh
go run github.com/googleapis/librarian/cmd/librarian@latest -help
```

### Setup and Configuration
To run librarian, you will need to configure a Github Token and set it to the `LIBRARIAN_GITHUB_TOKEN` environment variable. This will use the Github token as authentication to push to Github and to run Github API commands (e.g. Creating a PR and Adding Labels to a PR).

Unless specifically configured, Librarian will use HTTPS for pushing to remote. See the [SSH](#using-ssh) section for push to remote via SSH.

### Github Token
There are two main options to get a Github token:
1. Use the [gh cli](https://cli.github.com/) tool to easily authenticate with github. Run the following commands once the tool is installed:
```shell
# Follow the instructions from the tool. If using SSH, see the section
# below regarding additional setup.You still need a Github Token to run
# Github API commands (e.g. creating a pull request).
gh auth login 
# This will output the token to use as the Github Token
gh auth token
```
This will create a token with the `repo` scope.

2. Alternatively, follow the steps listed in the Github [guide](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens#creating-a-personal-access-token-classic) to create a Personal Access Token (PAT). You will need the `repo` scope for your token.

Once you have created a token, set the token as the environment variable:
```shell
export LIBRARIAN_GITHUB_TOKEN={YOUR_TOKEN_HERE}
```

### Using SSH
There are two main ways to configure SSH:
1. Using the [gh cli](https://cli.github.com/) to upload your SSH public key to Github. When following the steps in the gh cli tool, you will select your public key to be added to Github. Additionally, you will need to add your private key to the [ssh-agent](https://linux.die.net/man/1/ssh-agent).

Typically, your private keys will be `~/.ssh/id_ed25519` or `~/.ssh/id_rsa` and your public keys will be the same with the `.pub` suffix. You should be able to see this by running `ls ~/.ssh`. If you do not see a public/ private key combination, you can follow this [guide](https://docs.github.com/en/authentication/connecting-to-github-with-ssh/generating-a-new-ssh-key-and-adding-it-to-the-ssh-agent) to generate a new key.

Running the following commands to add your private key to ssh-agent:
```sh
# This will start the ssh-agent if it hasn't already been started
eval "$(ssh-agent -s)"

# This adds your private key to the ssh-agent.
# Note: The private key will not have the `.pub` suffix.
ssh-add ~/.ssh/{PRIVATE_KEY_FILE}

# Run this command to verify that your private key is added
# You should see an output of a SHA with an absolute path to the private key
# e.g. `256 SHA:{MY_SHA} .../.ssh/id_ed25519`
ssh-add -l
```

2. Follow the steps [here](https://docs.github.com/en/authentication/connecting-to-github-with-ssh). You will need to either create a new SSH key or using an existing one, add it to the ssh-agent, and then upload it to Github.

Once everything has been configured, set the SSH environment variable to push to Github remote with SSH:
```shell
export LIBRARIAN_USE_SSH=true
```

## Documentation

- [CLI Documentation](https://pkg.go.dev/github.com/googleapis/librarian/cmd/librarian)
- [Language Onboarding Guide](doc/language-onboarding.md)
- [How We Write Go](doc/howwewritego.md)
- [State Schema](doc/state-schema.md)
- [Config Schema](doc/config-schema.md)
- [Running Tests](doc/testing.md)
- [sidekick](doc/sidekick.md)

## License

Apache 2.0 - See [LICENSE](LICENSE) for more information.
