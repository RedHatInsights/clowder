#!/bin/bash

kubectl apply -f build/skuttl-namespace.yaml
kubectl apply -f build/skuttl-perms.yaml

kubectl kuttl test \
    --config bundle/tests/scorecard/kuttl/kuttl-test.yaml \
    --manifest-dir config/crd/bases/ \
    bundle/tests/scorecard/kuttl/ \
    $@  # pass in any extra cmd line args that you desire, such as "--test <test name>"
