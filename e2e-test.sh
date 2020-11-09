#!/bin/bash

export IMAGE_TAG=`git rev-parse --short HEAD`

kubectl apply -f skuttl-namespace.yaml
kubectl apply -f skuttl-perms.yaml

IMG=$IMAGE_NAME:$IMAGE_TAG make bundle
IMG=$IMAGE_NAME:$IMAGE_TAG make docker-build
IMG=$IMAGE_NAME:$IMAGE_TAG make docker-push
IMG=$IMAGE_NAME:$IMAGE_TAG make deploy
kubectl kuttl test --config bundle/tests/scorecard/kuttl/kuttl-test.yaml --crd-dir config/crd/bases/ bundle/tests/scorecard/kuttl/
#operator-sdk scorecard bundle --selector=suite=kuttlsuite --verbose --namespace=skuttl-test --service-account kuttl -w 300s
