#!/usr/bin/env python3

import yaml
import sys
import os
import argparse

yamls = yaml.safe_load_all(sys.stdin)

parser = argparse.ArgumentParser(description="Convert kustomize manifest to template")
parser.add_argument('--mutate', dest="mutate", action="store_true",
    help="flag to set if we should include the mutatingwebhookconfiguration")
parser.add_argument('--config', dest="config", action="store",
    help="supply a config file to use as the clowder-config configmap"
)
args = parser.parse_args()

with open("template.yml") as fp:
    template = yaml.safe_load(fp)

template["objects"].extend(yamls)

if not args.mutate:
    delete = []
    for i, object in enumerate(template["objects"]):
        if object["kind"] == "MutatingWebhookConfiguration":
            delete.append(i)

    for item in delete:
        del template["objects"][item]

if args.config:
    with open(os.path.realpath(args.config)) as f:
        data = f.read()
    config = yaml.safe_load(data)
    template["objects"].append(config)

print(yaml.dump(template))
