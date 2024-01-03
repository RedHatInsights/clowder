#!/bin/bash

set -e

python -m venv docs/build/venv
source docs/build/venv/bin/activate
pip install json-schema-for-humans==0.47
generate-schema-doc --config template_name=md controllers/cloud.redhat.com/config/schema.json docs/api_ref.md
