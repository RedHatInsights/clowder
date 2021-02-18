#!/bin/bash

set -e


if [ ! -d docs/build/crd-ref-docs ]; then
	echo "You don't have crd-ref-docs, installing it..."
	mkdir -p docs/build/crd-ref-docs
	git clone --branch psav/add_rst_template https://github.com/psav/crd-ref-docs.git docs/build/crd-ref-docs
	cd docs/build/crd-ref-docs
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

crd-ref-docs --source-path=./apis --config=docs/build/crd-ref-docs/config.yaml \
	--renderer=restructuredtext --templates-dir=docs/build/crd-ref-docs/templates/restructuredtext \
	--output-path=api_reference.rst

LINES_CHANGED=$(diff api_reference.rst docs/api_reference.rst --changed-group-format='%>' --unchanged-group-format='' | wc -l)

if [ "$LINES_CHANGED" == "0" ]; then
	echo "API docs did not change - not updating"
	rm api_reference.rst
else
	echo "API docs changed - updating"
	mv api_reference.rst docs/
fi
