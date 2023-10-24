#!/usr/bin/env python

import sys
import yaml

filename = sys.argv[1]
namespace = sys.argv[2]

print(f"Replacing: {filename} - {namespace}")

with open(filename, "r") as f:
    yaml_data = yaml.safe_load(f)
    for i, env in enumerate(yaml_data['spec']['template']['spec']['containers'][0]['env']):
        if env['name'] == "STRIMZI_NAMESPACE":
            try:
                del yaml_data['spec']['template']['spec']['containers'][0]['env'][i]['valueFrom']
            except KeyError:
                pass
            yaml_data['spec']['template']['spec']['containers'][0]['env'][i]['value'] = namespace

with open(filename, "w") as f:
    yaml.dump(yaml_data, f, indent=2)
