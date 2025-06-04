# Protos in Librarian

Librarian uses pipeline-state.json and pipeline-config.json to record
the state of the pipeline and some repo-specific configuration
aspects. Typically pipeline-state.json is maintained by Librarian (and
by the language container just within the 'configure' Librarian command)
whereas pipeline-config.json is expected to be maintained by hand (and
rarely change).

For both files, Librarian and language containers are expected to
agree on the format, so it's important to have a schema for those
files. The proto files within this directory form that schema.

Languages do not *have* to generate code based on these protos. In
particular, the files are in JSON, not in binary wire format. Where a
language finds it simpler to represent the same schema e.g. with
annotated source code, that's fine. However, it's also fine to add the
librarian repo as a submodule and generate source code based on the
protos here.

The `generate-proto.sh` script in the root directory of this repo is
used to generate the source code in the right place in this repo when
the protos are updated. (Currently the script has to be run manually.
The protos don't change often enough to make it worth automating that.)
