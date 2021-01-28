#!/bin/bash
set -e

export IMAGE_TAG=`git rev-parse --short HEAD`

kubectl apply -f build/skuttl-namespace.yaml
kubectl apply -f build/skuttl-perms.yaml
kubectl apply -f build/prommie-operator-bundle.yaml

export IMG=$IMAGE_NAME:$IMAGE_TAG

make bundle
make docker-build-no-test
make docker-push
make deploy
kubectl kuttl test --config bundle/tests/scorecard/kuttl/kuttl-test.yaml --crd-dir config/crd/bases/ bundle/tests/scorecard/kuttl/
#operator-sdk scorecard bundle --selector=suite=kuttlsuite --verbose --namespace=skuttl-test --service-account kuttl -w 300s
