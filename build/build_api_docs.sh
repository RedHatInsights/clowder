#!/bin/bash

set -e


if [ ! -d docs/build/crd-ref-docs ]; then
	echo "You don't have crd-ref-docs, installing it..."
	mkdir -p docs/build/crd-ref-docs
	git clone https://github.com/elastic/crd-ref-docs.git docs/build/crd-ref-docs
	cd docs/build/crd-ref-docs
	git checkout v0.0.12
	go build -o crd-ref-docs main.go
	cd -
fi

docs/build/crd-ref-docs/crd-ref-docs --source-path=./apis --config=docs/config.yaml \
	--renderer=markdown \
	--output-path=docs/api_reference.md
