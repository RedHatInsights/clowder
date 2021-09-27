#!/usr/bin/env python3

import yaml
import sys

yamls = yaml.safe_load_all(sys.stdin)

with open("template.yml") as fp:
    template = yaml.safe_load(fp)

template["objects"].extend(yamls)

print(yaml.dump(template))
