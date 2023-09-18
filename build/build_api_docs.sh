#!/bin/bash

set -e


if [ ! -d docs/build/crd-ref-docs ]; then
	echo "You don't have crd-ref-docs, installing it..."
	mkdir -p docs/build/crd-ref-docs
	git clone https://github.com/elastic/crd-ref-docs.git docs/build/crd-ref-docs
	cd docs/build/crd-ref-docs
	go build -o crd-ref-docs main.go
	cd -
fi

if ! command -v asciidoctor; then
	echo "ERROR: 'asciidoctor' not found."
	echo " "
	echo "Please install 'asciidoctor'.  On Fedora use 'sudo dnf install -y asciidoctor'"
	echo "For others, see: https://docs.asciidoctor.org/asciidoctor/latest/install/"
	exit 1
fi

docs/build/crd-ref-docs/crd-ref-docs --source-path=./apis --config=docs/config.yaml \
	--renderer=asciidoctor --templates-dir=docs/build/crd-ref-docs/templates/asciidoctor \
	--output-path=docs/antora/modules/ROOT/pages/api_reference.adoc
