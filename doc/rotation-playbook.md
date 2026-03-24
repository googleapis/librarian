# Librarian rotation playbook

As a librarian on-call, you need to perform library boarding, code generation, and release for SDKs that have onboarded
the librarian tool.

Each SDK has its own generation and release cadence. See each SDK section for more details.

## Keep on-call records
Create a new section in the [on-call notes](https://docs.google.com/document/d/1moP7zq3Qy7xqjdKSpTJhHbhpX7dSG9Wo38f9nlEpcLQ/edit?resourcekey=0-p1YPSwMTMGFYRgr4jegcOw&tab=t.eor0pn4ku93s)
by copying the template.
Update your LDAP and rotation dates.

Write down commands you need to execute during the rotation.
In addition, write down what you think are working well and what should be improved.

We can use this data to improve the rotation experience.

## Rust

### Regenerate Rust SDK

Regenerate SDK once a week.

#### Install protoc
1. Check the [protoc version](https://github.com/googleapis/google-cloud-rust/blob/main/.gcb/scripts/regenerate.sh#L19)
in GCB workflow.
2. Download the protoc binary from [protobuf releases](https://github.com/protocolbuffers/protobuf/releases).
3. Run `protoc --version` to verify the version.

#### Update librarian version
```shell
# In google-cloud-rust repository root directory, assuming you are using a fork.
git checkout main
git fetch upstream
git merge --ff-only upstream/main
git checkout -b chore-update-librarian-$(date +%Y-%m-%d)
V=$(GOPROXY=direct go list -m -f '{{.Version}}' github.com/googleapis/librarian@main)
sed -i.bak "s;^version: .*;version: ${V};" librarian.yaml && rm librarian.yaml.bak
git add . && git commit -m "chore: update librarian version"
```
Create a pull request and merge it.

#### Update source and regenerate all Rust libraries
```shell
# In google-cloud-rust repository root directory, assuming you are using a fork.
git checkout main
git fetch upstream
git merge --ff-only upstream/main
git checkout -b chore-update-libraries-$(date +%Y-%m-%d)
V=$(sed -n 's/^version: *//p' librarian.yaml)
# Update sources
go run github.com/googleapis/librarian/cmd/librarian@${V} update discovery
go run github.com/googleapis/librarian/cmd/librarian@${V} update googleapis
# Regenerate
go run github.com/googleapis/librarian/cmd/librarian@${V} generate --all
git add . && git commit -m "chore: update sources and regenerate"
```
Create a pull request and merge it.

### Release

Release Rust SDK once a month, subscribe [SDK release calendar](https://calendar.google.com/calendar/u/0?cid=Y184ZmNlZjVkZmUwMzM1NTFhNjg2ZTU4MWY2NWRlYTMyZjIxMDcxZWQyNDNkZjBkYWViNGViMzAzZjJkMWI4NzM4QGdyb3VwLmNhbGVuZGFyLmdvb2dsZS5jb20)
for more details.
You don't need to release Rust SDK if no release scheduled during your rotation period.

Follow instructions in go/cloud-rust:release-playbook