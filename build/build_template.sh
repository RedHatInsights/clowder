#!/bin/bash

python3 -m venv "build/.build_venv"
source build/.build_venv/bin/activate
pip install pyyaml

if [ -f "deploy-mutate.yml" ]; then
  mv deploy-mutate.yml deploy-mutate.yml.old
fi

if [ -f "deploy.yml" ]; then
  mv deploy.yml deploy.yml.old
fi
cat $TEMPLATE_KUSTOMIZE | ./manifest2template.py --config config/deployment-template/clowder_config.yaml --mutate > deploy-mutate.yml
cat $TEMPLATE_KUSTOMIZE | ./manifest2template.py --config config/deployment-template/clowder_config.yaml > deploy.yml
deactivate
