#!/usr/bin/env python

import sys
import ruamel.yaml

filename = sys.argv[1]
namespace = sys.argv[2]

yaml = ruamel.yaml.YAML()

print(f"Replacing: {filename} - {namespace}")

with open(filename, "r") as f:
    yaml_data = ruamel.yaml.round_trip_load(f)
    for i, env in enumerate(yaml_data['spec']['template']['spec']['containers'][0]['env']):
        if env['name'] == "STRIMZI_NAMESPACE":
            try:
                del yaml_data['spec']['template']['spec']['containers'][0]['env'][i]['valueFrom']
            except KeyError:
                pass
            yaml_data['spec']['template']['spec']['containers'][0]['env'][i]['value'] = namespace

with open(filename, "w") as f:
    ruamel.yaml.round_trip_dump(yaml_data, f, indent=2)
