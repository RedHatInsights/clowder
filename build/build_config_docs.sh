#!/bin/bash

set -e


if [ ! -f docs/build/node_modules/.bin/jsonschema2md ]; then
	echo "You don't have jsonschema2md, installing it..."
	cd docs/build/
	if [ ! -d node_modules ]; then
		mkdir node_modules
	fi
	npm install @adobe/jsonschema2md@4.2.2
	cd -
fi


if [ ! -f docs/build/node_modules/.bin/jsonschema2md ]; then
	echo "ERROR: Could not find jsonschema2md, please check installation above"
	exit 1
fi

./docs/build/node_modules/.bin/jsonschema2md -d controllers/cloud.redhat.com/config/ -o docs/appconfig -e json

rm out/schema.json
rmdir out

## Lets move to using coveooss/json-schema-for-humans
