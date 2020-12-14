#!/bin/bash

export IMAGE_TAG=`git rev-parse --short HEAD`

kubectl apply -f skuttl-namespace.yaml
kubectl apply -f skuttl-perms.yaml
kubectl apply -f prommie-operator-bundle.yaml

export IMG=$IMAGE_NAME:$IMAGE_TAG

make bundle
make docker-build-no-test
make docker-push-local
make deploy
kubectl kuttl test --config bundle/tests/scorecard/kuttl/kuttl-test.yaml --crd-dir config/crd/bases/ bundle/tests/scorecard/kuttl/
#operator-sdk scorecard bundle --selector=suite=kuttlsuite --verbose --namespace=skuttl-test --service-account kuttl -w 300s
