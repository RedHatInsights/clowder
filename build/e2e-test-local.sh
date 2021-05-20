#!/bin/bash
set -e

bash build/kube_setup.sh
make deploy-minikube-quick
bash build/run_kuttl.sh $@  # pass any cli options to kuttl, such as "--test <test name>"

#operator-sdk scorecard bundle --selector=suite=kuttlsuite --verbose --namespace=skuttl-test --service-account kuttl -w 300s
