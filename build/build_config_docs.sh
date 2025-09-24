#!/bin/bash

set -e

python3 -m venv docs/build/venv
source docs/build/venv/bin/activate
pip install json-schema-for-humans==1.4.1
generate-schema-doc --config with_footer=false --config template_name=md controllers/cloud.redhat.com/config/schema.json docs/api_ref.md
