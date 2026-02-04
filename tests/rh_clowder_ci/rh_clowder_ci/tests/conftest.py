import logging
import os
from importlib.resources import files
from typing import Dict, List

import pytest
import yaml
from bonfire.openshift import wait_for_all_resources
from ocviapy import apply_config, get_api_resources, oc, process_template

logger = logging.getLogger(__name__)

DEFAULT_NAMESPACE = os.environ.get("TEST_NS", "clowder-e2e-test")


@pytest.fixture(scope="session")
def namespace() -> str:
    """Provide a default namespace for tests."""
    return DEFAULT_NAMESPACE


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


def cleanup_resources(resources: List[Dict[str, str]], namespace: str = None):
    """
    Intelligently clean up resources based on whether they are namespaced.

    For namespaced resources, deleting the namespace will clean them up.
    For cluster-scoped resources, explicitly delete them.
    """
    logger.info("Cleaning up resources created by the test")
    namespace = namespace or DEFAULT_NAMESPACE

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
        logger.info("Deleting cluster-scoped resource: %s/%s", kind, name)
        try:
            oc("delete", kind, name, "--ignore-not-found=true")
        except Exception as e:
            logger.info("Failed to delete %s/%s: %s", kind, name, e)

    # Delete the namespace (which will clean up all namespaced resources)
    logger.info("Deleting namespace: %s", namespace)
    try:
        oc("delete", "namespace", namespace, "--wait=true", "--ignore-not-found=true")
    except Exception as e:
        logger.info("Failed to delete namespace %s: %s", namespace, e)


@pytest.fixture(scope="module")
def deploy_test_resources():
    created_resources = []
    ns = DEFAULT_NAMESPACE

    def _deploy_test_resources(
            resource_file_name: str,
            namespace: str = None,
            wait_timeout: int = 600
        ):
        """Deploy test resources and wait for them to be ready."""
        nonlocal created_resources
        nonlocal ns
        ns = namespace  # set 'ns' for cleanup use

        # Load template from package resources
        # Note: importlib.resources is the modern standard library replacement for pkg_resources
        template_content = (
            files("rh_clowder_ci.resources")
            .joinpath(resource_file_name)
            .read_text()
        )
        template_dict = yaml.safe_load(template_content)

        # Ensure namespace exists
        try:
            oc("get", "namespace", namespace, _silent=True)
        except Exception:
            oc("create", "namespace", namespace)

        # Process template
        logger.info("Processing template and applying to namespace: %s", namespace)
        processed_content = process_template(template_dict, {"NAMESPACE": namespace})

        # Parse resources from processed content for cleanup tracking
        created_resources = _parse_resources(processed_content)

        # Apply resources to namespace
        apply_config(namespace, processed_content)

        # Wait for all resources to be ready
        logger.info("Waiting for resources to be ready...")
        wait_for_all_resources(namespace, timeout=wait_timeout)

        logger.info("Resources deployed and ready")

    try:
        yield _deploy_test_resources
    finally:
        # Clean up resources intelligently
        cleanup_resources(created_resources, ns)
