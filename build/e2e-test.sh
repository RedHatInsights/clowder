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

bash build/run_kuttl.sh

#operator-sdk scorecard bundle --selector=suite=kuttlsuite --verbose --namespace=skuttl-test --service-account kuttl -w 300s
