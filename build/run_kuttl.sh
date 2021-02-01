#!/bin/bash

kubectl kuttl test \
    --config bundle/tests/scorecard/kuttl/kuttl-test.yaml \
    --manifest-dir config/crd/bases/ \
    --manifest-dir config/crd/static/ \
    bundle/tests/scorecard/kuttl/
