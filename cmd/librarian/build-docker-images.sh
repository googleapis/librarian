#!/bin/sh

# This script accepts a librarian version (e.g.
# "v0.8.4-0.20260308144746-c8a4cb903c10") and builds local Docker images
# (one per language). The version number is split on "-" and the 3rd section
# is used as a commit hash to check out before building.
#
# Currently, this script should be run from the cmd/librarian directory.

set -e

if [[ $1 == "" ]]
then
  echo "Must specify a version, e.g. v0.8.4-0.20260308144746-c8a4cb903c10"
  exit 1
fi

commit=$(echo $1 | cut -d- -f3)
original_branch=$(git branch --show-current)

if [[ $commit == "" || $commit == $1 ]]
then
  echo "Invalid version; no commit"
  exit 1
fi

echo "Checking out $commit"
git checkout $commit

# TODO(https://github.com/googleapis/librarian/issues/4467) Build for all
# languages when the Dockerfile is fixed.
for language in python
do
  (cd ../.. && docker build -t librarian-$language:$1 -f cmd/librarian/Dockerfile --target $language .)
done

if [[ $original_branch != "" ]]
then
  echo "Checking out $original_branch"
  git checkout $original_branch
fi
