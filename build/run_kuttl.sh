#!/bin/bash

export ARTIFACTS_DIR=${ARTIFACTS_DIR:-"$(pwd)/artifacts"}

echo "kuttl test artifacts will be saved to: ${ARTIFACTS_DIR}"

kubectl kuttl test \
    --config tests/kuttl/kuttl-test.yaml \
    --manifest-dir config/crd/bases/ \
    --artifacts-dir $ARTIFACTS_DIR \
    tests/kuttl/ \
    $@  # pass in any extra cmd line args that you desire, such as "--test <test name>"
