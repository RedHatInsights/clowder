import json
import os
from importlib.resources import files
from typing import Dict, List

import pytest
import yaml
from bonfire.openshift import wait_for_all_resources
from ocviapy import apply_config, get_api_resources, get_json, oc, process_template
from wait_for import wait_for


def _parse_resources(processed_content: dict) -> List[Dict[str, str]]:
    """Parse processed template content and extract resource metadata (kind, name, namespace)."""
    resources = []

    for item in processed_content.get("items", []):
        if not item or not isinstance(item, dict):
            continue

        kind = item.get("kind")
        name = item.get("metadata", {}).get("name")
        namespace = item.get("metadata", {}).get("namespace")

        if kind and name:
            resources.append({
                "kind": kind,
                "name": name,
                "namespace": namespace
            })

    return resources


def _cleanup_resources(resources: List[Dict[str, str]], namespace: str):
    """
    Intelligently clean up resources based on whether they are namespaced.

    For namespaced resources, deleting the namespace will clean them up.
    For cluster-scoped resources, explicitly delete them.
    """
    print("Cleaning up resources created by the test")

    # Get API resources info to determine which resources are namespaced
    api_resources = get_api_resources()

    # Build a lookup map: kind (lowercase) -> namespaced (bool)
    namespaced_map = {}
    for resource in api_resources:
        kind = resource.get("kind", "").lower()
        namespaced = resource.get("namespaced", False)
        namespaced_map[kind] = namespaced

    # Separate resources into namespaced and cluster-scoped
    cluster_scoped_resources = []

    for resource in resources:
        kind = resource["kind"]
        name = resource["name"]

        # Check if this resource type is namespaced
        is_namespaced = namespaced_map.get(kind.lower(), True)  # Default to True

        if not is_namespaced:
            cluster_scoped_resources.append(resource)

    # Delete cluster-scoped resources first
    for resource in cluster_scoped_resources:
        kind = resource["kind"]
        name = resource["name"]
        print(f"Deleting cluster-scoped resource: {kind}/{name}")
        try:
            oc("delete", kind, name, "--ignore-not-found=true")
        except Exception as e:
            print(f"Failed to delete {kind}/{name}: {e}")

    # Delete the namespace (which will clean up all namespaced resources)
    print(f"Deleting namespace: {namespace}")
    try:
        oc("delete", "namespace", namespace, "--wait=true", "--ignore-not-found=true")
    except Exception as e:
        print(f"Failed to delete namespace {namespace}: {e}")


@pytest.fixture(scope="module")
def deploy_test_resources():
    """Deploy ClowdApp test resources and wait for them to be ready."""
    namespace = os.environ.get("TEST_NS", "clowder-e2e")
    wait_timeout = os.environ.get("WAIT_TIMEOUT", "5m")

    # Load template from package resources
    # Note: importlib.resources is the modern standard library replacement for pkg_resources
    template_content = (
        files("rh_clowder_ci.resources")
        .joinpath("puptoo-test-resources.yaml")
        .read_text()
    )
    template_dict = yaml.safe_load(template_content)

    # Ensure namespace exists
    try:
        oc("get", "namespace", namespace, _silent=True)
    except Exception:
        oc("create", "namespace", namespace)

    # Process template
    print(f"Processing template and applying to namespace: {namespace}")
    processed_content = process_template(template_dict, {"NAMESPACE": namespace})

    # Parse resources from processed content for cleanup tracking
    created_resources = _parse_resources(processed_content)

    # Apply resources to namespace
    apply_config(namespace, processed_content)

    try:
        # Wait for all resources to be ready
        print("Waiting for resources to be ready...")
        wait_for_all_resources(namespace, timeout=wait_timeout)

        print("Resources deployed and ready")

        # Yield to run tests
        yield

    finally:
        # Clean up resources intelligently
        _cleanup_resources(created_resources, namespace)


def _get_cdappconfig_json(namespace: str, app_name: str) -> dict:
    secret = get_json("secret", app_name, namespace=namespace)
    raw_b64 = secret.get("data", {}).get("cdappconfig.json", "")
    assert raw_b64, "cdappconfig.json not found in secret data"
    from base64 import b64decode
    content = b64decode(raw_b64).decode("utf-8")
    return json.loads(content)

def test_cdappconfig_secret_exists(deploy_test_resources):
    namespace = os.environ.get("TEST_NS", "clowder-e2e")
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

def test_cdappconfig_content_exact_match(deploy_test_resources):
    namespace = os.environ.get("TEST_NS", "clowder-e2e")
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
