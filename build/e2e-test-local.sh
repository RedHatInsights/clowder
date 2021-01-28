#!/bin/bash
set -e

kubectl apply -f build/skuttl-namespace.yaml
kubectl apply -f build/skuttl-perms.yaml
kubectl apply -f build/prommie-operator-bundle.yaml

make deploy-minikube
kubectl kuttl test --config bundle/tests/scorecard/kuttl/kuttl-test.yaml --crd-dir config/crd/bases/ bundle/tests/scorecard/kuttl/
#operator-sdk scorecard bundle --selector=suite=kuttlsuite --verbose --namespace=skuttl-test --service-account kuttl -w 300s
