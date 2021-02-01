#!/bin/bash
set -e

kubectl apply -f build/skuttl-namespace.yaml
kubectl apply -f build/skuttl-perms.yaml
kubectl apply -f build/prommie-operator-bundle.yaml

make deploy-minikube

bash build/run_kuttl.sh

#operator-sdk scorecard bundle --selector=suite=kuttlsuite --verbose --namespace=skuttl-test --service-account kuttl -w 300s
