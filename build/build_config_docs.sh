#!/bin/bash

set -e

python3 -m venv docs/build/venv
source docs/build/venv/bin/activate
pip install json-schema-for-humans==v1.0.2
generate-schema-doc --config with_footer=false --config template_name=md controllers/cloud.redhat.com/config/schema.json docs/api_ref.md
