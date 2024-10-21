#!/bin/bash

kuttl test \
    --config tests/kuttl/kuttl-test.yaml \
    --manifest-dir config/crd/bases/ \
    tests/kuttl/ \
    $@  # pass in any extra cmd line args that you desire, such as "--test <test name>"
