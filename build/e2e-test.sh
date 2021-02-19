#!/bin/bash
set -e

export IMAGE_TAG=`git rev-parse --short HEAD`

bash build/kube_setup.sh

export IMG=$IMAGE_NAME:$IMAGE_TAG

make bundle
make docker-build-no-test
make docker-push
make deploy

bash build/run_kuttl.sh $@  # pass any cli options to kuttl, such as "--test <test name>"

#operator-sdk scorecard bundle --selector=suite=kuttlsuite --verbose --namespace=skuttl-test --service-account kuttl -w 300s
