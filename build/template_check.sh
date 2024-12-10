#!/bin/bash

set -exv

python3 -m venv "build/.build_venv"
source build/.build_venv/bin/activate
pip install pyyaml

CURRENT_DEPLOY=$(md5sum deploy.yml)
CURRENT_MUTATE=$(md5sum deploy-mutate.yml)

make build-template

if [[ $CURRENT_DEPLOY != $(md5sum deploy.yml) ]]; then
    echo "Deployment template not updated. Please run make build-template and recommit"
    exit 1
else
    echo "Deployment template is up to date"
fi

if [[ $CURRENT_MUTATE != $(md5sum deploy-mutate.yml) ]]; then
    echo "Mutating template not updated. Please run make build-template and recommit"
    exit 1
else
    echo "Mutating template is up to date"
fi

deactivate
