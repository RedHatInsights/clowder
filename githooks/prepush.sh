#!/bin/bash

function prepush() {
make pre-push

# If the make command failed, prevent the push
if [[ -n "$(git status -s)" ]]; then
  echo "ERROR: Please run 'make pre-push', then 'git add' any changes, and then 'git push' a new commit."
  exit 1
fi
}

prepush
