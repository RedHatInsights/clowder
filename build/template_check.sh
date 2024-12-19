#!/bin/bash

set -exv

if diff -q deploy.yml.old deploy.yml > /dev/null; then
    echo "Deployment template is up to date"
else
    echo "Deployment template [deploy.yml] not updated. Please run make build-template and recommit"
    exit 1
fi

if diff -q deploy-mutate.yml.old deploy-mutate.yml > /dev/null; then
    echo "Deployment template is up to date"
else
    echo "Deployment template [deploy-mutate.yml] not updated. Please run make build-template and recommit"
    exit 1
fi
