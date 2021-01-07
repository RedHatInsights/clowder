#!/bin/bash

set -e


if [ ! -d .crd-ref-docs ]; then
	echo "You don't have crd-ref-docs, installing it..."
	mkdir .crd-ref-docs
	git clone https://github.com/elastic/crd-ref-docs.git .crd-ref-docs
	cd .crd-ref-docs
	go install
	cd -
fi


if ! command -v crd-ref-docs; then
	echo "ERROR: could not find 'crd-ref-docs', check if 'go install' succeeded or 'rm -rf .crd-ref-docs' and try again"
	exit 1
fi


if ! command -v asciidoctor; then
	echo "ERROR: 'asciidoctor' not found."
	echo " "
	echo "Please install 'asciidoctor'.  On Fedora use 'sudo dnf install -y asciidoctor'"
	echo "For others, see: https://docs.asciidoctor.org/asciidoctor/latest/install/"
	exit 1
fi

crd-ref-docs --source-path=./apis --config=.crd-ref-docs/config.yaml \
	--renderer=asciidoctor --templates-dir=.crd-ref-docs/templates/asciidoctor \
	--output-path=api_reference.asciidoc

asciidoctor api_reference.asciidoc

rm api_reference.asciidoc

mv api_reference.html docs/

