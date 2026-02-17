import json
import os

from ocviapy import get_json
from wait_for import wait_for


def _get_cdappconfig_json(namespace: str, app_name: str) -> dict:
    secret = get_json("secret", app_name, namespace=namespace)
    raw_b64 = secret.get("data", {}).get("cdappconfig.json", "")
    assert raw_b64, "cdappconfig.json not found in secret data"
    from base64 import b64decode
    content = b64decode(raw_b64).decode("utf-8")
    return json.loads(content)


def test_cdappconfig_secret_exists(deploy_test_resources, namespace):
    deploy_test_resources("puptoo-test-resources.yaml")

    app_name = os.environ.get("CLOWDAPP_NAME", "puptoo")

    def _secret_has_cdappconfig():
        try:
            secret = get_json("secret", app_name, namespace=namespace)
            return bool(secret.get("data", {}).get("cdappconfig.json"))
        except Exception:
            return False

    # Wait up to 5 minutes for Secret to appear
    wait_for(
        _secret_has_cdappconfig,
        timeout=300,
        delay=5,
        message=f"cdappconfig secret {app_name} not found or missing data in ns {namespace}"
    )

    # Validate it parses as JSON via helper
    _ = _get_cdappconfig_json(namespace, app_name)


def test_cdappconfig_content_exact_match(deploy_test_resources, namespace):
    deploy_test_resources("puptoo-test-resources.yaml")

    app_name = os.environ.get("CLOWDAPP_NAME", "puptoo")

    expected = {
        "endpoints": [
            {
                "apiPath": "/api/puptoo-processor/",
                "apiPaths": ["/api/puptoo-processor/"],
                "app": "puptoo",
                "h2cPort": 0,
                "h2cTLSPort": 0,
                "hostname": f"puptoo-processor.{namespace}.svc",
                "name": "processor",
                "port": 8000,
                "tlsPort": 0,
            }
        ],
        # hashCache is intentionally not asserted (can vary); removed before comparison
        "hashCache": "IGNORED",
        "logging": {
            "cloudwatch": {
                "accessKeyId": "",
                "logGroup": "",
                "region": "",
                "secretAccessKey": "",
            },
            "type": "null",
        },
        "metadata": {
            "deployments": [
                {"image": "quay.io/psav/clowder-hello", "name": "processor"}
            ],
            "envName": "test-basic-app",
            "name": "puptoo",
        },
        "metricsPath": "/metrics",
        "metricsPort": 9000,
        "privateEndpoints": [
            {
                "app": "puptoo",
                "h2cPort": 0,
                "h2cTLSPort": 0,
                "hostname": f"puptoo-processor.{namespace}.svc",
                "name": "processor",
                "port": 10000,
                "tlsPort": 0,
            }
        ],
        "privatePort": 10000,
        "publicPort": 8000,
        "webPort": 8000,
    }
    actual = _get_cdappconfig_json(namespace, app_name)
    # Ignore hashCache differences
    actual.pop("hashCache", None)
    expected.pop("hashCache", None)
    assert actual == expected, (
        "cdappconfig mismatch.\nExpected: "
        + json.dumps(expected, sort_keys=True)
        + "\nActual: "
        + json.dumps(actual, sort_keys=True)
    )
