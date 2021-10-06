#!/usr/bin/env python3

import yaml
import sys

yamls = yaml.safe_load_all(sys.stdin)

with open("template.yml") as fp:
    template = yaml.safe_load(fp)

template["objects"].extend(yamls)

if "--mutate" not in sys.argv:
    delete = []
    for i, object in enumerate(template["objects"]):
        if object["kind"] == "MutatingWebhookConfiguration":
            delete.append(i)

    for item in delete:
        del template["objects"][item]

print(yaml.dump(template))
