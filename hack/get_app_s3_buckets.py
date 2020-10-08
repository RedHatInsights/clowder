#!/usr/bin/env python3

import json

from graphqlclient import GraphQLClient
from collections import defaultdict

query = """
{
  ns: namespaces_v1 {
    name
    environment {
      name
    }
    terraformResources {
      provider
      output_resource_name
      identifier
    }
    app {
      name
      parentApp {
        name
      }
    }
  }
}
"""

client = GraphQLClient("http://localhost:4000/graphql")
namespaces = json.loads(client.execute(query))["data"]["ns"]

buckets = defaultdict(list)

db_list = []

for ns in namespaces:
    if ns["app"]["parentApp"] is None or ns["app"]["parentApp"]["name"] != "insights":
        continue

#     if ns["environment"]["name"] != "insights-production":
#         continue

    for terraform in ns["terraformResources"] or []:
        if terraform["provider"] == "rds" and "prod" in terraform['identifier']:
            buckets[ns['app']['name']].append(f"{terraform['identifier']} -> {terraform['output_resource_name']}")
            db_list.append(terraform['identifier'])

max_app = max(len(a) for a in buckets)

for app in buckets:
    for bucket in sorted(buckets[app]):
        print(f"{(app + ':').ljust(max_app + 1)} {bucket}")

for bucket in sorted(db_list):
    if bucket.split('-')[-1] not in ("ci", "qa", "stage", "prod"):
        print(bucket)
