import os
import json
import shutil
import subprocess
import time

import pytest


def _run(cmd: list[str]) -> subprocess.CompletedProcess:
    return subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, check=False)


def _get_yaml_value(file_path: str, yq_query: str) -> str:
    # Try yq v4 if present; otherwise fallback to grep-like parsing
    if shutil.which("yq"):
        out = _run(["yq", "e", yq_query, file_path])
        if out.returncode == 0:
            return out.stdout.strip()
    # naive fallback: look for name under metadata; tests can also rely on env overrides
    return ""


def _get_env_and_app_from_resources() -> tuple[str | None, str | None]:
    res = "/workspace/resources.yaml"
    if not os.path.exists(res):
        return None, None
    # Attempt via oc get from file
    # First ClowdEnvironment name
    ce = _run(["oc", "get", "-f", res, "-o", "jsonpath={..metadata.name}"])
    ce_name = ce.stdout.strip() if ce.returncode == 0 else ""
    # First ClowdApp name
    ca = _run(["oc", "get", "-f", res, "-o", "jsonpath={..metadata.name}"])
    ca_name = ca.stdout.strip() if ca.returncode == 0 else ""
    return (ce_name or None, ca_name or None)


@pytest.mark.timeout(600)
def test_cdappconfig_secret_exists():
    namespace = os.environ.get("TEST_NS", "clowder-e2e")
    app_name = os.environ.get("CLOWDAPP_NAME")

    if not app_name:
        _, app_name = _get_env_and_app_from_resources()

    assert app_name, "CLOWDAPP_NAME not provided and could not infer from resources.yaml"

    # Wait up to 5 minutes for Secret to appear
    secret_name = app_name  # convention: secret name equals app name
    for _ in range(60):
        cp = _run(["oc", "get", "secret", secret_name, "-n", namespace, "-o", "jsonpath={.data.cdappconfig\\.json}"])
        if cp.returncode == 0 and cp.stdout.strip():
            break
        time.sleep(5)
    else:
        pytest.fail(f"cdappconfig secret {secret_name} not found or missing data in ns {namespace}")

    # Basic sanity: can we decode the JSON content?
    # Use oc to decode base64
    cp_json = _run(["sh", "-lc", f"oc get secret {secret_name} -n {namespace} -o jsonpath='{{.data.cdappconfig\\.json}}' | base64 -d"])
    assert cp_json.returncode == 0, cp_json.stderr
    assert cp_json.stdout.strip(), "cdappconfig.json content is empty"

    # Validate it parses as JSON
    json.loads(cp_json.stdout)
